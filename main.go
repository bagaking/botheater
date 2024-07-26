package main

import (
	"context"
	"github.com/bagaking/botheater/playground/theater"
	"github.com/sirupsen/logrus"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/call/tool"
	"github.com/bagaking/botheater/tools"
	"github.com/bagaking/botheater/utils"
	"github.com/bagaking/goulp/wlog"
)

var tm = tool.NewToolManager()

func main() {
	utils.MustInitLogger()
	logrus.SetLevel(logrus.TraceLevel)

	ctx := context.Background()
	logger := wlog.ByCtx(context.Background())
	logger.Infof("start botheater playground ...")

	tm.RegisterTool(&tools.LocalFileReader{})
	tm.RegisterTool(&tools.RandomIdeaGenerator{})
	tm.RegisterTool(&tools.GoogleSearcher{})
	tm.RegisterTool(&tools.Browser{})

	conf := LoadConf(ctx)

	botLoader := bot.NewBotLoader(tm).LoadBots(ctx, conf.bots)

	theater.Play(ctx, botLoader)
}

//
//func TestNormalChat(ctx context.Context, b *bot.Bot, question string) {
//	log := wlog.ByCtx(ctx, "TestNormalChat")
//	h := history.NewHistory()
//	resp, err := b.Question(ctx, h, question)
//	if err != nil {
//		log.WithError(err).Errorf("chat failed")
//	}
//
//	log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", resp)
//}
//
//func TestContinuousChat(ctx context.Context, b *bot.Bot) {
//	log := wlog.ByCtx(ctx, "TestContinuousChat")
//	reader := bufio.NewReader(os.Stdin)
//
//	for {
//		fmt.Print("Enter your question: ")
//		question, _ := reader.ReadString('\n')
//		question = question[:len(question)-1] // 去掉换行符
//
//		h := history.NewHistory()
//		got, err := b.Question(ctx, h, question)
//		if err != nil {
//			log.WithError(err).Errorf("chat failed")
//			continue
//		}
//
//		log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", got)
//
//	}
//}

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
