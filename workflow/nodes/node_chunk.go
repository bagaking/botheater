package nodes

import (
	"context"
	"strings"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/utils"
	"github.com/bagaking/botheater/workflow"
)

type (
	ChunkSizeContext interface{ GetChunkSize() int }
)

const (
	LoopUpperBound = 999
)

var _ workflow.NodeExecutor = SplitOriginTextIntoChunks[interface{ GetChunkSize() int }]

// SplitOriginTextIntoChunks
// in - input
// out - chunks
func SplitOriginTextIntoChunks[TCtx ChunkSizeContext](ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	logger := wlog.ByCtx(ctx, "SplitOriginTextIntoChunks")
	finished := false

	input, _input := "", params["input"]
	if str, ok := _input.(string); !ok {
		return "", irr.Error("input param is not set")
	} else {
		input = str
	}

	// 从 context 中获取 workflowCtx
	wfCtx, _ := workflow.CtxValue[TCtx](ctx)
	size := typer.Or(wfCtx.GetChunkSize(), 1024*8)

	logger.Infof("chunk size is %d, using %d, content tokens is %v", wfCtx.GetChunkSize(), size, utils.CountTokens(input))

	chunks := splitOriginTextIntoChunks(ctx, input, size)

	wlog.ByCtx(ctx, "SplitOriginTextIntoChunks").Infof("chunkResult len= %d", len(chunks))
	if finished, err = signal(ctx, "chunks", chunks); err != nil {
		return "", err
	}

	if !finished {
		return "", irr.Error("node is not finish")
	}
	return "success", nil
}

func splitOriginTextIntoChunks(ctx context.Context, input string, size int) []string {
	ret := make([]string, 0, len(input)/size)
	for para, left := utils.TakeSentences(strings.Split(input, "\n"), size); para != nil; para, left = utils.TakeSentences(left, size) {
		ret = append(ret, string(para))
	}
	return ret
}

func splitOriginBytesIntoChunks(ctx context.Context, input string, size int) []string {
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
