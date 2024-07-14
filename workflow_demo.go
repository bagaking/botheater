package main

import (
	"context"
	"github.com/bagaking/botheater/workflow"
	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
	"strings"
)

type workflowCtx struct {
	ChunkSize int
}

var (
	_ workflow.NodeExecutor = SplitOriginTextIntoChunks
)

// SplitOriginTextIntoChunks
// in - input
// out - chunks
func SplitOriginTextIntoChunks(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	finished := false

	input, _input := "", params["input"]
	if str, ok := _input.(string); !ok {
		return "", irr.Error("input param is not set")
	} else {
		input = str
	}
	chunks := splitOriginTextIntoChunks(ctx, input)
	if finished, err = signal(ctx, "chunks", chunks); err != nil {
		return "", err
	}

	if !finished {
		return "", irr.Error("node is not finish")
	}
	return "success", nil
}

func splitOriginTextIntoChunks(ctx context.Context, input string) []string {
	// 从 context 中获取 workflowCtx
	wfCtx, _ := workflow.CtxValue[workflowCtx](ctx)

	size := typer.Or(wfCtx.ChunkSize, 1024*8)

	var list []string

	// 暴力分段
	for i := 0; i < 3200; i++ {
		if len(input) > size {
			list = append(list, input[:size])
			input = input[size:]
			continue
		}
		if len(input) > size/2 {
			list = append(list, input)
			break
		}
		if len(list) > 0 {
			list[len(list)-1] += input
		} else {
			list = append(list, input)
		}
		break
	}
	return list
}

func TryWorkflow(ctx context.Context) {
	log := wlog.ByCtx(ctx, "TestWorkflow")
	wf := workflow.New("vc_summary")

	// 定义起始节点和结束节点
	if err := wf.SetStartNode([]string{"text"}); err != nil {
		log.WithError(err).Errorf("set start node failed")
		return
	}
	wf.SetEndNode([]string{"result"})

	nodeA := workflow.NewNode("SplitOriginTextIntoChunks", SplitOriginTextIntoChunks, []string{"input"}, []string{"chunks"})

	if err := workflow.Connect(wf.StartNode, "text", nodeA, "input"); err != nil {
		log.Fatalf("连接起始节点到 NodeA 失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "chunks", wf.EndNode, "result"); err != nil {
		log.Fatalf("连接 NodeA 到终止节点失败: %v", err)
	}

	longText := strings.Repeat("0123456789ABCDEFG", 1024)

	// 执行工作流
	outTable, err := wf.Execute(context.Background(), workflow.ParamsTable{"text": longText})
	if err != nil {
		log.Fatalf("工作流执行失败: %v", err)
	}
	if !wf.Finished() {
		log.Fatalf("工作流执行异常")
	}

	result, ok := outTable["result"].([]string)
	if !ok {
		log.Fatalf("工作流执行结果类型错误")
	}

	log.Infof("result len= %d, data= %d", len(result), typer.SliceMap(result, func(x string) int { return len(x) }))
}
