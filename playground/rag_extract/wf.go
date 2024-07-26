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
	ExtractEntity   *bot.Bot `bot:"rag_extract_entity"`
	ExtractRelation *bot.Bot `bot:"rag_extract_relation"`
	MergeEntity     *bot.Bot `bot:"rag_merge_entity"`
}

func TryWorkflow(ctx context.Context, loader *bot.Loader) {
	wfCtx := workflowCtx{
		ChunkSize: 3 * 1024,
	}
	ctx = workflow.WithCtx(ctx, wfCtx)
	logger := wlog.ByCtx(ctx, "TryWorkflow")

	use := UsingBots{}
	if err := loader.StaplingBots(ctx, &use); err != nil {
		logger.Fatalf("stapling bots failed, err= %v", err)
	}

	wf := workflow.New("rag_test")

	// 定义起始节点和结束节点
	if err := wf.SetStartNode([]string{"text"}); err != nil {
		logger.WithError(err).Errorf("set start node failed")
		return
	}
	wf.SetEndNode([]string{"entities", "relations"})

	nodeChunk := workflow.NewNode("SplitOriginTextIntoChunks", nodes.SplitOriginTextIntoChunks[workflowCtx], []string{"input"}, []string{"chunks"})

	unmarshal2StrLst := func(answer string) (any, error) {
		if answer == "" {
			return nil, irr.Error("answer is empty")
		}
		lst := make([]any, 0)
		if err := jsonex.Unmarshal([]byte(answer), &lst); err != nil {
			return nil, irr.Wrap(err, "unmarshal tidy answer to list failed")
		}
		return lst, nil
	}

	nodeBotExtractEntity := workflow.NewNodeByDef(nodes.NewBotWorkflowNode(use.ExtractEntity, unmarshal2StrLst))
	nodeBotExtractRelation := workflow.NewNodeByDef(nodes.NewBotWithHistoryWorkflowNode(use.ExtractRelation, unmarshal2StrLst))
	nodeBotMergeEntity := workflow.NewNodeByDef(nodes.NewBotReduceWorkflowNode(use.MergeEntity, unmarshal2StrLst))

	conn := &workflow.Connector{}
	{
		nodeMap := map[string]workflow.Node{
			"n_start": wf.StartNode,
			"n_end":   wf.EndNode,

			"_serializer_lst": workflow.NewNodeByDef(nodes.NewSerializerNode(nodes.SerializeModeJsonStrLst)),
			"_serializer_str": workflow.NewNodeByDef(nodes.NewSerializerNode(nodes.SerializeModeYamlStr)),
			"collect_1":       workflow.NewNodeByDef(nodes.NewCollectWorkflowNode([]string{"in1"}, "")),
			"collect_2":       workflow.NewNodeByDef(nodes.NewCollectWorkflowNode([]string{"in1", "in2"}, "")),
			"n_print":         workflow.NewNodeByDef(nodes.NewPrinterWorkflowNode("text")),
			"n_flatten": workflow.NewNodeByDef(nodes.NewMergeSliceWorkflowNode(), workflow.WithEventCallback(func(event, log string) {
				wlog.Common("Flatten").Infof("%s : %s", event, log)
			})),

			"n_chunks":             nodeChunk,
			"bot_extract_entity":   nodeBotExtractEntity,
			"bot_extract_relation": nodeBotExtractRelation,
			"bot_merge_entity":     nodeBotMergeEntity,
		}

		conn.Use(ctx, nodeMap, Script1)
	}
	if err := conn.Error(); err != nil {
		logger.Fatalf("connect failed, err= %v", err)
	}

	// 执行工作流
	outTable, err := wf.Execute(ctx, workflow.ParamsTable{"text": Text4Test})
	if err != nil {
		logger.Fatalf("工作流执行失败: %v", err)
	}
	if !wf.Finished() {
		logger.Fatalf("工作流执行异常")
	}

	entities, ok := outTable["entities"].(string)
	if !ok {
		logger.Fatalf("工作流执行结果 entities 类型错误")
	}

	relations, ok := outTable["relations"].(string)
	if !ok {
		logger.Fatalf("工作流执行结果 relations 类型错误")
	}

	logger.Infof("entities len= %d, token= %d", len(entities), utils.CountTokens(entities))
	logger.Infof("relations len= %d, token= %d", len(relations), utils.CountTokens(relations))

	logger.Infof("\n\n%s", utils.SPrintWithFrameCard("抽取的实体", entities, 168, utils.StyConclusion))
	logger.Infof("\n\n%s", utils.SPrintWithFrameCard("这些实体的关系", relations, 168, utils.StyConclusion))
}

const Script1 = `%%{init: {'theme':'base',"fontFamily": "monospace", "sequence": { "wrap": true }, "flowchart": { "curve": "linear" } }}%%
flowchart TD

n_start --> n_chunks[n_chunks:将结果先分段] --> bot_extract_entity([bot_extract_entity\n逐个chunk抽取实体]) -- _serializer_lst --> bot_merge_entity([bot_merge_entity\n合并相同实体]) 

bot_merge_entity -- _serializer_str |:history| --> bot_extract_relation([bot_extract_relation\n逐个chunk抽取实体关系内容])
n_chunks -->|:question| bot_extract_relation

bot_merge_entity -- n_flatten, _serializer_str |:entities| --> n_end((结束))
bot_extract_relation -- n_flatten, _serializer_str |:relations| --> n_end
`
