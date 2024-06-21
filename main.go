// Usage:
//
// 1. go get -u github.com/volcengine/volc-sdk-golang
// 2. VOLC_ACCESSKEY=XXXXX VOLC_SECRETKEY=YYYYY go run main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/bagaking/botheater/history"

	"github.com/bagaking/goulp/jsonex"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/tool"
	"github.com/bagaking/botheater/tools"
	"github.com/bagaking/goulp/wlog"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
)

const (
	BotNameBasic    = "botheater_basic"
	BotNameFunction = "botheater_function"
)

var tm = tool.NewToolManager()

func main() {
	logrus.SetLevel(logrus.TraceLevel)

	log, ctx := wlog.ByCtxAndCache(context.Background())

	tm.RegisterTool(&tools.LocalFileReader{})
	tm.RegisterTool(&tools.RandomIdeaGenerator{})

	conf := LoadConf(ctx)
	b, err := conf.NewBot(initClient(ctx), BotNameBasic, tm)
	// b, err := conf.NewBot(initClient(ctx), BotNameFunction)
	if err != nil {
		log.WithError(err).Fatalf("create b failed")
	}

	botFileReader, err := conf.NewBot(initClient(ctx), "botheater_filereader", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_coordinator failed")
	}

	botCoordinator, err := conf.NewBot(initClient(ctx), "botheater_coordinator", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_coordinator failed")
	}

	bot.InitAllActAs(ctx, botCoordinator, botFileReader, b)

	log.Info(b.String())

	// TestNormalChat(ctx, b, "给我一个好点子")
	// TestNormalChat(ctx, b, "阅读当前目录下的关键代码内容后，找到和处理 req.Messages & b history 有关的代码，并提取出一个队列来对其进行优化。给我这个队列的代码")
	// TestContinuousChat(ctx, b)
	// TestStreamChat(ctx, b, req)

	h := history.NewHistory()
	MultiAgentChat(ctx, h, botCoordinator, "接下来我要对本地仓库代码做优化，准备好了吗？") //
	MultiAgentChat(ctx, h, botFileReader, "阅读当前目录下的关键代码内容后，找到和处理 req.Messages & b history 有关的代码，并提取出一个队列来对其进行优化。给我这个队列的代码")
	MultiAgentChat(ctx, h, botCoordinator, "总结之前聊天里，你的观点, 以及用于佐证的代码") //
	MultiAgentChat(ctx, h, b, "针对这些代码进行改写，使其更优雅，要注意不要重复造轮子")
}

func TestNormalChat(ctx context.Context, b *bot.Bot, question string) {
	log := wlog.ByCtx(ctx, "TestNormalChat")
	h := history.NewHistory()
	resp, err := b.NormalChat(ctx, h, question)
	if err != nil {
		log.WithError(err).Errorf("chat failed")
	}

	log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", bot.Resp2Str(resp))
}

func MultiAgentChat(ctx context.Context, h *history.History, b *bot.Bot, question string) {
	log := wlog.ByCtx(ctx, "TestNormalChat")
	resp, err := b.NormalChat(ctx, h, question)
	if err != nil {
		log.WithError(err).Errorf("chat failed")
	}

	log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", bot.Resp2Str(resp))
	h.EnqueueAssistantMsg(bot.RespMsg2Str(resp))
}

func TestContinuousChat(ctx context.Context, b *bot.Bot) {
	log := wlog.ByCtx(ctx, "TestContinuousChat")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your question: ")
		question, _ := reader.ReadString('\n')
		question = question[:len(question)-1] // 去掉换行符

		h := history.NewHistory()
		resp, err := b.NormalChat(ctx, h, question)
		if err != nil {
			log.WithError(err).Errorf("chat failed")
			continue
		}

		log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", bot.Resp2Str(resp))

	}
}

func TestStreamChat(ctx context.Context, b *bot.Bot, question string) {
	log := wlog.ByCtx(ctx, "TestNormalChat")

	handler := func(resp *api.ChatResp) {
		if resp.Error != nil {
			// it is possible that error occurs during response processing
			log.Info(jsonex.MustMarshalToString(resp.Error))
			return
		}
		log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", bot.Resp2Str(resp))
	}

	h := history.NewHistory()
	if err := b.StreamChat(ctx, h, question, handler); err != nil {
		log.WithError(err).Errorf("chat failed")
	}
}
