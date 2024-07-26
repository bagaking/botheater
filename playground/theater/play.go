package theater

import (
	"context"
	"fmt"
	"strings"

	"github.com/bagaking/botheater/utils"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/history"
)

const (
	MaxRound        = 100000
	ContinueMessage = `如果达到目标了请回答 "任务完成"，并对整个聊天进行总结后，对用户的原始问题进行正式答复; 否则, 进一步分析接下来该做什么，并说明步骤`
)

func Play(ctx context.Context, loader *bot.Loader) {
	logger := wlog.ByCtx(ctx, "play_theater")
	//TestNormalChat(ctx, botBasic, "给我一个好点子")
	//TestNormalChat(ctx, botBasic, "阅读当前目录下的关键代码内容后，找到和处理 req.Messages & botBasic history 有关的代码，并提取出一个队列来对其进行优化。给我这个队列的代码")
	//TestContinuousChat(ctx, botBasic)
	//TestStreamChat(ctx, botBasic, req)

	h := history.NewHistory()
	//MultiAgentChat(ctx, h, "接下来我要对本地仓库代码做优化，准备好了吗？", botCoordinator) //
	//
	//MultiAgentChat(ctx, h,
	//	"阅读当前目录下的关键代码内容后，找到和处理 req.Messages & botBasic history 有关的代码，并提取出一个队列来对其进行优化。给我这个队列的代码",
	//	botCoordinator, botFileReader, botBasic)
	//
	//MultiAgentChat(ctx, h, "帮我找到比特币最近的行情", botCoordinator, botFileReader, botBasic) // 搜索可能要优化
	//MultiAgentChat(ctx, h, "帮我总结什么是鸟狗式", bots...)
	//
	//MultiAgentChat(ctx, h, "什么是vector_database", bots...)

	bots, err := loader.GetBots()
	if err != nil {
		logger.Fatalf("Failed to load bots: %v", err)
	}
	bot.InitActAsForBots(ctx, bots...)
	MultiAgentChat(ctx, h, "找到现在这个本地仓库 util 里在带框架卡片具体实现原理和用法，然后参照任意 github 的 readme 格式，写一份 README.md 介绍功能的原理和具体用法", bots...)
	//theater.MultiAgentChat(ctx, h, "找到现在这个本地仓库 util 里在把文字 format 成卡片格式的原理具体实现原理和用法，然后参照任意 github 的 readme 格式，写一份 README.md 介绍功能的原理和具体用法", bots...)
	//theater.MultiAgentChat(ctx, h, "现在这个本地仓库中的框架，能够确保 agent 很好的调用 functions，看看这是怎么做到的？这里的设计有什么独到之处？然后上网看看有没有类似的实现", bots...)
	//theater.MultiAgentChat(ctx, h, "找到现在这个本地仓库中 bot 的实现代码，然后对 bot 的实现思路进行总结", bots...)
	//theater.MultiAgentChat(ctx, h, "总结之前聊天里，你的观点, 以及用于佐证的代码", botCoordinator) //
	//theater.MultiAgentChat(ctx, h, "针对这些代码进行改写，使其更优雅，要注意不要重复造轮子", botBasic)

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

	answer := ""
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
			h.EnqueueAssistantMsg("chat failed, err: "+err.Error(), bCur.PrefabName)
		} else {
			h.EnqueueAssistantMsg(content, bCur.PrefabName)
		}

		log.Infof("=== chat answer round %d ===\n\n%s=== chat answer ===\n\n", i+1, content)
		answer = content
		// continue chat
		if bot.Caller.HasCall(content) { // 任何有指名 agent 的情况，都调用
			agentName, params, err := bot.Caller.ParseCall(ctx, content)
			log.Debugf("agent call found, name= %s, params= %v", agentName, params)
			if err != nil {
				log.WithError(err).Errorf("chat continue stop at round %d, parse call failed", i)
				break
			}

			q := strings.Join(params, "; ")

			ind := typer.SliceFirstMatch(bots, func(b *bot.Bot) bool {
				return b.PrefabName == agentName
			})
			if ind < 0 { // 找不到 agent 的情况以为着 coordinator 判断出错, 或者 5xx 了
				log.Errorf("cannot find next agent, call failed %s", agentName)
				break
			}

			// 设置下次执行的 agent
			bCur = bots[ind]

			// 如果上一句话是 bCoordinate 的决策过程，那么把决策过程删除，只保留谁来做的结论和任务
			// todo：可能不把 history 中的决策过程踢出去更好
			if last, ok := h.PeekTail(); ok && bCoordinate != nil && last.Identity == bCoordinate.PrefabName {
				t, _ := h.PopTail() // Pop 栈头
				log.Infof("dequeue coordinate agent msg, content= %v", t)
				if last, ok = h.PeekTail(); ok && last.Role == history.RoleUser && last.Content == ContinueMessage {
					t, _ = h.PopTail() // Pop 栈头
					log.Infof("dequeue coordinate user msg, content= %v", t)
				}

				h.EnqueueCoordinateMsg(
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

	log.Infof("\n%s\n",
		utils.SPrintWithFrameCard("CHAT ANSWER", answer, utils.PrintWidthL1, utils.StyConclusion))
}
