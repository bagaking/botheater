package nodes

import (
	"context"

	"strings"

	"github.com/bagaking/botheater/utils"
	"github.com/bagaking/botheater/workflow"
	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
)

type (
	ChunkSizeContext interface{ GetChunkSize() int }

	ChunkMethod func(ctx context.Context, input string, size int) []string

	WFChunkNode[T ChunkSizeContext] struct {
		Method ChunkMethod
	}
)

const (
	InNameChunk  = "input"
	OutNameChunk = "chunks"

	LoopUpperBound   = 999
	DefaultChunkSize = 1024 * 8
)

var _ workflow.NodeDef = &WFChunkNode[ChunkSizeContext]{}

func NewChunkNode[T ChunkSizeContext](method ChunkMethod) *WFChunkNode[T] {
	ret := &WFChunkNode[T]{
		Method: method,
	}
	return ret
}

// Execute - SplitTextIntoChunks
// in - input
// out - chunks
func (n *WFChunkNode[T]) Execute(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	logger := wlog.ByCtx(ctx, "chunk")
	finished := false

	input, _input := "", params[InNameChunk]
	if str, ok := _input.(string); !ok {
		return "", irr.Error("input param is not set")
	} else {
		input = str
	}

	// 从 context 中获取 workflowCtx
	wfCtx, _ := workflow.CtxValue[T](ctx)
	size := typer.Or(wfCtx.GetChunkSize(), DefaultChunkSize)

	fn := n.Method
	if fn == nil {
		fn = splitOriginTextIntoChunks
	}
	logger.Infof("chunk size is %d, using %d, content tokens is %v", wfCtx.GetChunkSize(), size, utils.CountTokens(input))

	chunks := fn(ctx, input, size)

	wlog.ByCtx(ctx, "SplitTextIntoChunks").Infof("chunkResult len= %d", len(chunks))
	if finished, err = signal(ctx, OutNameChunk, chunks); err != nil {
		return "", err
	}

	if !finished {
		return "", irr.Error("node is not finish")
	}
	return "success", nil
}

func (n *WFChunkNode[T]) Name() string { return "chunk" }

func (n *WFChunkNode[T]) InNames() []string {
	return []string{InNameChunk}
}

func (n *WFChunkNode[T]) OutNames() []string {
	return []string{OutNameChunk}
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
