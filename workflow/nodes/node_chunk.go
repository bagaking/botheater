package nodes

import (
	"context"

	"github.com/bagaking/goulp/wlog"

	"github.com/bagaking/botheater/workflow"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
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
	wlog.ByCtx(ctx, "splitOriginTextIntoChunks").Infof("chunk size is %d", wfCtx.ChunkSize)

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
		}
		break
	}
	return list
}
