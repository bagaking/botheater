package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/bagaking/goulp/jsonex"
	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/workflow"
)

type (
	// WFCollect 收集多个 input 到一个 collect 中
	// collect 默认是一个 []any，当某个 input 也是 []any 时，则会默认展开
	WFCollect struct {
		inNames []string
		outMode any
	}
)

const (
	OutCollect = "collect"
)

var _ workflow.NodeDef = &WFCollect{}

func NewCollectWorkflowNode(inNames []string, outMode any) *WFCollect {
	switch outMode.(type) {
	case []any, []string, string:
	default:
		panic(irr.Error("unsupported out mode %v", outMode))
	}
	return &WFCollect{inNames: inNames, outMode: outMode}
}

func (n *WFCollect) Execute(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	finished := false
	defer func() {
		if err == nil && !finished {
			err = irr.Error("node is not finish")
		}
	}()

	inputs := typer.Vals(params)
	wlog.ByCtx(ctx, "collect").Infof("get all inputs key %v", typer.Keys(params))
	sliceFlatten := typer.FlattenNestedSlices(inputs, 2)
	wlog.ByCtx(ctx, "collect").Infof("collect flatten result,\n- form: %v - to: %v", len(inputs), len(sliceFlatten))

	typer.SliceForeachI(sliceFlatten, func(v any, i int) {
		wlog.ByCtx(ctx, "collect").Infof("get all inputs flatten val %d. `%v`", i, v)
	})

	switch n.outMode.(type) {
	case []any:
		finished, err = signal(ctx, OutCollect, sliceFlatten)
	case []string:
		out := typer.SliceMap(sliceFlatten, func(v any) string {
			if v == nil {
				return ""
			}
			if v.(string) != "" {
				return v.(string)
			}
			return jsonex.MustMarshalToString(v)
		})
		finished, err = signal(ctx, OutCollect, out)
	case string:
		out := jsonex.MustMarshalToString(sliceFlatten)
		finished, err = signal(ctx, OutCollect, out)
	default:
		return "", irr.Error("unsupported out mode %v", n.outMode)
	}

	return "success, slice merged", nil
}

func (n *WFCollect) Name() string {
	return fmt.Sprintf("collect(%s)", strings.Join(n.inNames, ","))
}

func (n *WFCollect) InNames() []string {
	return n.inNames
}

func (n *WFCollect) OutNames() []string {
	return []string{OutCollect}
}
