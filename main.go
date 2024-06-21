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

	log.Info(b.String())

	// TestNormalChat(ctx, b, "给我一个好点子")
	TestNormalChat(ctx, b, "阅读当前目录下的关键代码内容后，找到和处理 req.Messages & b history 有关的代码，并提取出一个队列来对其进行优化。给我这个队列的代码")
	// TestContinuousChat(ctx, b)
	// TestStreamChat(ctx, b, req)
}

func TestNormalChat(ctx context.Context, b *bot.Bot, question string) {
	log := wlog.ByCtx(ctx, "TestNormalChat")
	resp, err := b.NormalChat(ctx, question)
	if err != nil {
		log.WithError(err).Errorf("chat failed")
	}

	log.Infof("=== chat answer ===\n%s", bot.Resp2Str(resp))
}

func TestContinuousChat(ctx context.Context, b *bot.Bot) {
	log := wlog.ByCtx(ctx, "TestContinuousChat")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your question: ")
		question, _ := reader.ReadString('\n')
		question = question[:len(question)-1] // 去掉换行符

		resp, err := b.NormalChat(ctx, question)
		if err != nil {
			log.WithError(err).Errorf("chat failed")
			continue
		}

		log.Infof("=== chat answer ===\n%s", bot.Resp2Str(resp))

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
		log.Info(jsonex.MustMarshalToString(resp))
		// last response may contain `usage`
		if resp.Usage != nil {
			// last message, will return full response including usage, role, finish_reason, etc.
			log.Info(jsonex.MustMarshalToString(resp.Usage))
		}
	}

	if err := b.StreamChat(ctx, question, handler); err != nil {
		log.WithError(err).Errorf("chat failed")
	}
}
