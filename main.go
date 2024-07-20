package main

import (
	"context"
	"github.com/bagaking/botheater/wf_rag"

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
	log := wlog.ByCtx(context.Background())

	tm.RegisterTool(&tools.LocalFileReader{})
	tm.RegisterTool(&tools.RandomIdeaGenerator{})
	tm.RegisterTool(&tools.GoogleSearcher{})
	tm.RegisterTool(&tools.Browser{})

	conf := LoadConf(ctx)

	botCoordinator, err := conf.NewBot(ctx, "botheater_coordinator", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_coordinator failed")
	}

	botBasic, err := conf.NewBot(ctx, "botheater_basic", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botBasic failed")
	}

	botFileReader, err := conf.NewBot(ctx, "botheater_filereader", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_coordinator failed")
	}

	botFileSearcher, err := conf.NewBot(ctx, "botheater_filesearcher", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_searcher failed")
	}

	botSearcher, err := conf.NewBot(ctx, "botheater_searcher", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_searcher failed")
	}

	botCodeReader, err := conf.NewBot(ctx, "botheater_codereader", tm)
	if err != nil {
		log.WithError(err).Fatalf("create botheater_codereader failed")
	}

	botRagExtractEntity, err := conf.NewBot(ctx, "rag_extract_entity", tm)
	if err != nil {
		log.WithError(err).Fatalf("create rag_extract_entity failed")
	}

	botRagMergeEntity, err := conf.NewBot(ctx, "rag_merge_entity", tm)
	if err != nil {
		log.WithError(err).Fatalf("create rag_merge_entity failed")
	}

	botRagExtractRelation, err := conf.NewBot(ctx, "rag_extract_relation", tm)
	if err != nil {
		log.WithError(err).Fatalf("create rag_extract_relation failed")
	}

	bots := []*bot.Bot{
		botCoordinator, botBasic, botSearcher, botFileSearcher, botFileReader, botCodeReader,
	}

	bot.InitAllActAs(ctx, bots...)

	log.Info(botBasic.String())

	wf_rag.TryWorkflow(ctx, wf_rag.UsingBots{
		ExtractEntity:   botRagExtractEntity,
		ExtractRelation: botRagExtractRelation,
		MergeEntity:     botRagMergeEntity,
	})
}
