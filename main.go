package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/call/tool"
	"github.com/bagaking/botheater/history"
	"github.com/bagaking/botheater/tools"
	"github.com/bagaking/botheater/utils"
)

const (
	MaxRound        = 100000
	ContinueMessage = "达到目标了吗? 接下来该做什么？"
)

var tm = tool.NewToolManager()

func main() {
	utils.MustInitLogger()
	logrus.SetLevel(logrus.TraceLevel)

	ctx := context.Background()
	log := wlog.ByCtx(context.Background())

	tm.RegisterTool(&tools.LocalFileReader{})
	tm.RegisterTool(&tools.RandomIdeaGenerator{})
	tm.RegisterTool(&tools.GoogleSearcher{})
	tm.RegisterTool(&tools.Browser{})

	conf := LoadConf(ctx)

	botBasic, err := conf.NewBot(ctx, "botheater_basic", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botBasic failed")
	}

	botFileReader, err := conf.NewBot(ctx, "botheater_filereader", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_coordinator failed")
	}

	botCoordinator, err := conf.NewBot(ctx, "botheater_coordinator", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_coordinator failed")
	}

	botSearcher, err := conf.NewBot(ctx, "botheater_searcher", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_searcher failed")
	}

	bots := []*bot.Bot{
		botCoordinator, botFileReader, botBasic, botSearcher,
	}

	bot.InitAllActAs(ctx, bots...)

	log.Info(botBasic.String())

	// TestNormalChat(ctx, botBasic, "给我一个好点子")
	// TestNormalChat(ctx, botBasic, "阅读当前目录下的关键代码内容后，找到和处理 req.Messages & botBasic history 有关的代码，并提取出一个队列来对其进行优化。给我这个队列的代码")
	// TestContinuousChat(ctx, botBasic)
	// TestStreamChat(ctx, botBasic, req)

	h := history.NewHistory()
	// MultiAgentChat(ctx, h, "接下来我要对本地仓库代码做优化，准备好了吗？", botCoordinator) //

	// MultiAgentChat(ctx, h,
	//	"阅读当前目录下的关键代码内容后，找到和处理 req.Messages & botBasic history 有关的代码，并提取出一个队列来对其进行优化。给我这个队列的代码",
	//	botCoordinator, botFileReader, botBasic)

	// MultiAgentChat(ctx, h, "帮我找到比特币最近的行情", botCoordinator, botFileReader, botBasic) // 搜索可能要优化
	// MultiAgentChat(ctx, h, "帮我总结什么是鸟狗式", bots...)

	MultiAgentChat(ctx, h, "什么是vector_database", bots...)
	// MultiAgentChat(ctx, h, "总结之前聊天里，你的观点, 以及用于佐证的代码", botCoordinator) //
	// MultiAgentChat(ctx, h, "针对这些代码进行改写，使其更优雅，要注意不要重复造轮子", botBasic)
}

func MultiAgentChat(ctx context.Context, h *history.History, question string, bots ...*bot.Bot) {
	l2, ctx := wlog.ByCtxAndRemoveCache(ctx, "MultiAgentChat")
	log := l2.WithField("mode", "auto")

	if len(bots) == 0 {
		return
	}
	bCur := bots[0]
	var bCoordinate *bot.Bot
	for i := range bots {
		b := bots[i]
		if b.AckAs == bot.ActAsCoordinator {
			bCoordinate = b
			break
		}
	}
	if bCoordinate != nil {
		bCur = bCoordinate
	}

	h.EnqueueUserMsg(question)

	for i := 0; i < MaxRound; i++ {
		log = log.WithField("round", i)
		if bCur == nil {
			log.Errorf("bCur cannot be nil")
			return
		}
		log.Infof("enter new round= %d", i)
		content, err := bCur.SendChat(ctx, h)
		if err != nil {
			log.WithError(err).Errorf("chat failed")
		}

		h.EnqueueAssistantMsg(content, bCur.PrefabName)
		log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", content)

		// continue chat
		if bot.Caller.HasCall(content) { // 任何有指名 agent 的情况，都调用
			agentName, params, err := bot.Caller.ParseCall(ctx, content)
			log.Debugf("agent call found, name= %s, params= %v", agentName, params)
			if err != nil {
				log.WithError(err).Errorf("chat continue stop at round %d, parse call failed", i)
				break
			}

			if ind := typer.SliceFirstMatch(bots, func(b *bot.Bot) bool {
				return b.PrefabName == agentName
			}); ind >= 0 {
				bCur = bots[ind]
				q := strings.Join(params, "; ")
				// todo：可能不把 history 中的决策过程踢出去更好
				if last, ok := h.PeekTail(); ok && bCoordinate != nil && last.Identity == bCoordinate.PrefabName { // 如果上一句话是 bCoordinate 的决策过程
					t, _ := h.PopTail() // Pop 栈头
					log.Infof("dequeue coordinate agent msg, content= %v", t)
					if last, ok = h.PeekTail(); ok && last.Role == history.RoleUser && last.Content == ContinueMessage {
						t, _ = h.PopTail() // Pop 栈头
						log.Infof("dequeue coordinate user msg, content= %v", t)
					}
					h.EnqueueAssistantMsg(
						fmt.Sprintf("%s 经过思考，决定接下来 agent::%s 来做:\n %s", bCoordinate.PrefabName, bCur.PrefabName, q),
						bCoordinate.PrefabName,
					)

					//
					//// 如果只是删除这个消息会导致后边判断 Coordinator 的逻辑出错，改成以 coord 身份提出
					//if bot.Caller.HasCall(q) {
					//
					//} else {
					//	h.EnqueueAssistantMsg(bCoordinate.PrefabName+" 想到要"+q+"但还要想想调用哪个 agent", bCoordinate.PrefabName)
					//}
				} else {
					h.EnqueueUserMsg(q)
				}

				log.Infof("find next agent, %s(%s)", agentName, q)
				continue
			}

			log.Errorf("cannot find next agent, call failed %s", agentName)
			break
		}

		// 没有 agent 指名
		if bCoordinate == nil { // 如果又没有协调者就退出
			log.WithError(err).Warnf("chat continue stop at round %d, has no call and no coordinator", i)
			break
		}

		// 如果连续 Coordinator (上次 bCur == bCoordinate)，并且也没指定任何 agent 干活，那说明死循环了，再问也意思不大，应该用户干预
		if bCoordinate == bCur {
			log.Warnf("chat continue stop at round %d, coordinator has no idea", i)
			break
		}

		h.EnqueueUserMsg(ContinueMessage)
		bCur = bCoordinate
		log.Infof("back to coordinate")
	}
}

func TestNormalChat(ctx context.Context, b *bot.Bot, question string) {
	log := wlog.ByCtx(ctx, "TestNormalChat")
	h := history.NewHistory()
	resp, err := b.Question(ctx, h, question)
	if err != nil {
		log.WithError(err).Errorf("chat failed")
	}

	log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", resp)
}

func TestContinuousChat(ctx context.Context, b *bot.Bot) {
	log := wlog.ByCtx(ctx, "TestContinuousChat")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your question: ")
		question, _ := reader.ReadString('\n')
		question = question[:len(question)-1] // 去掉换行符

		h := history.NewHistory()
		got, err := b.Question(ctx, h, question)
		if err != nil {
			log.WithError(err).Errorf("chat failed")
			continue
		}

		log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", got)

	}
}

//func TestStreamChat(ctx context.Context, b *bot.Bot, question string) {
//	log := wlog.ByCtx(ctx, "TestNormalChat")
//
//	handler := func(resp *api.ChatResp) {
//		if resp.Error != nil {
//			// it is possible that error occurs during response processing
//			log.Info(jsonex.MustMarshalToString(resp.Error))
//			return
//		}
//		log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", coze.Resp2Str(resp))
//	}
//
//	h := history.NewHistory()
//	if err := b.StreamChat(ctx, h, question, handler); err != nil {
//		log.WithError(err).Errorf("chat failed")
//	}
//}
