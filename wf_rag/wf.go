package wf_rag

import (
	"context"

	"github.com/bagaking/goulp/jsonex"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/utils"
	"github.com/bagaking/botheater/workflow"
	"github.com/bagaking/botheater/workflow/nodes"
	"github.com/bagaking/goulp/wlog"
)

type workflowCtx struct {
	ChunkSize int
}

func (w workflowCtx) GetChunkSize() int {
	return w.ChunkSize
}

type UsingBots struct {
	ExtractEntity   *bot.Bot
	ExtractRelation *bot.Bot
}

func TryWorkflow(ctx context.Context, bots UsingBots) {
	wfCtx := workflowCtx{
		ChunkSize: 2 * 1024,
	}
	ctx = workflow.WithCtx(ctx, wfCtx)

	log := wlog.ByCtx(ctx, "TryWorkflow")

	wf := workflow.New("rag_test")

	// 定义起始节点和结束节点
	if err := wf.SetStartNode([]string{"text"}); err != nil {
		log.WithError(err).Errorf("set start node failed")
		return
	}
	wf.SetEndNode([]string{"entities", "relations"})

	nodeChunk := workflow.NewNode("SplitOriginTextIntoChunks", nodes.SplitOriginTextIntoChunks[workflowCtx], []string{"input"}, []string{"chunks"})

	unmarshal2StrLst := func(answer string) (any, error) {
		lst := make([]any, 0)
		if err := jsonex.Unmarshal([]byte(answer), &lst); err != nil {
			return nil, irr.Wrap(err, "unmarshal tidy answer to list failed")
		}
		return lst, nil
	}

	nodeBotExtractEntity := workflow.NewNodeByDef(nodes.NewBotWorkflowNode(bots.ExtractEntity, unmarshal2StrLst))
	nodeBotExtractRelation := workflow.NewNodeByDef(nodes.NewBotWithHistoryWorkflowNode(bots.ExtractRelation, unmarshal2StrLst))

	conn := &workflow.Connector{}
	{
		nodeMap := map[string]workflow.Node{
			"n_start": wf.StartNode,
			"n_end":   wf.EndNode,

			"collect_1": workflow.NewNodeByDef(nodes.NewCollectWorkflowNode([]string{"in1"}, "")),
			"collect_2": workflow.NewNodeByDef(nodes.NewCollectWorkflowNode([]string{"in1", "in2"}, "")),
			"n_print":   workflow.NewNodeByDef(nodes.NewPrinterWorkflowNode("text")),
			"n_flatten": workflow.NewNodeByDef(nodes.NewMergeSliceWorkflowNode(), workflow.WithEventCallback(func(event, log string) {
				wlog.Common("Flatten").Infof("%s : %s", event, log)
			})),

			"n_chunks":             nodeChunk,
			"bot_extract_entity":   nodeBotExtractEntity,
			"bot_extract_relation": nodeBotExtractRelation,
		}

		conn.Use(ctx, nodeMap, Script1)
	}
	if err := conn.Error(); err != nil {
		log.Fatalf("connect failed, err= %v", err)
	}

	// 执行工作流
	outTable, err := wf.Execute(ctx, workflow.ParamsTable{"text": Text4Test})
	if err != nil {
		log.Fatalf("工作流执行失败: %v", err)
	}
	if !wf.Finished() {
		log.Fatalf("工作流执行异常")
	}

	entities, ok := outTable["entities"].([]any)
	if !ok {
		log.Fatalf("工作流执行结果 entities 类型错误")
	}

	e, err := jsonex.MarshalIndent(entities, "", "  ")
	if err != nil {
		log.Fatalf("工作流执行结果 entities 解析错误")
	}
	eStr := string(e)

	relations, ok := outTable["relations"].([]any)
	if !ok {
		log.Fatalf("工作流执行结果 relations 类型错误")
	}
	r, err := jsonex.MarshalIndent(relations, "", "  ")
	if err != nil {
		log.Fatalf("工作流执行结果 relations 解析错误")
	}
	rStr := string(r)

	log.Infof("entities len= %d, token= %d", len(entities), utils.CountTokens(eStr))
	log.Infof("relations len= %d, token= %d", len(relations), utils.CountTokens(rStr))

	log.Infof("\n\n%s", utils.SPrintWithFrameCard("抽取的实体", eStr, 168, utils.StyConclusion))
	log.Infof("\n\n%s", utils.SPrintWithFrameCard("这些实体的关系", rStr, 168, utils.StyConclusion))
}

const Script1 = `%%{init: {'theme':'base',"fontFamily": "monospace", "sequence": { "wrap": true }, "flowchart": { "curve": "linear" } }}%%
flowchart TD

n_start --> n_chunks[n_chunks:将结果先分段] --> bot_extract_entity([bot_extract_entity\n逐个chunk抽取实体]) 

bot_extract_entity -- n_flatten|:entities| --> n_end((结束))

bot_extract_entity -- n_flatten|:history| --> bot_extract_relation([bot_extract_relation\n逐个chunk抽取实体关系内容])

n_chunks -->|chunks:question| bot_extract_relation -- n_flatten|:relations| --> n_end((结束)) 

%% bot_extract_entity -- n_print --> __0
`
