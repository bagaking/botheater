package nodes

import (
	"context"
	"github.com/bagaking/botheater/utils"

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
	wlog.ByCtx(ctx, "SplitOriginTextIntoChunks").Infof("chunkResult len= %d", len(chunks))
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

	wlog.ByCtx(ctx, "splitOriginTextIntoChunks").Infof(
		"chunk size is %d, using %d, content tokens is %v", wfCtx.ChunkSize, size, utils.CountTokens(input))

	var list []string
	// 使用 rune 计算字符数
	runes := []rune(input)

	// 暴力分段
	for i := 0; i < 3200; i++ {
		if len(runes) > size {
			tokenCount := utils.CountTokens(string(runes[:size]))
			for tokenCount > size {
				size--
				tokenCount = utils.CountTokens(string(runes[:size]))
			}
			list = append(list, string(runes[:size]))
			runes = runes[size:]
			continue
		}
		if len(runes) > size/2 {
			list = append(list, string(runes))
			break
		}
		if len(list) > 0 {
			list[len(list)-1] += string(runes)
		}
		break
	}
	return list
}

func splitOriginBytesIntoChunks(ctx context.Context, input string) []string {
	// 从 context 中获取 workflowCtx
	wfCtx, _ := workflow.CtxValue[workflowCtx](ctx)

	size := typer.Or(wfCtx.ChunkSize, 1024*8)

	wlog.ByCtx(ctx, "splitOriginTextIntoChunks").Infof("chunk size is %d, using %d, content size is %v", wfCtx.ChunkSize, size, len(input))

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
