package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/call/tool"
	"github.com/bagaking/botheater/history"
	"github.com/bagaking/botheater/tools"
	"github.com/bagaking/botheater/utils"
	"github.com/bagaking/goulp/wlog"
)

const (
	MaxRound        = 100000
	ContinueMessage = "如果达到目标了请回答 `任务完成`，否则进一步分析接下来该做什么？"
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
