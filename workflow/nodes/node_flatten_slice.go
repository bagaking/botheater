package nodes

import (
	"context"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/workflow"
)

type WFFlatten2DSlice struct{}

const (
	InNameFlattenSlice  = "slices"
	OutNameFlattenSlice = "slice"
)

var _ workflow.NodeDef = &WFFlatten2DSlice{}

func NewMergeSliceWorkflowNode() *WFFlatten2DSlice {
	return &WFFlatten2DSlice{}
}

func (n *WFFlatten2DSlice) Execute(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	finished := false
	defer func() {
		if err == nil && !finished {
			err = irr.Error("node is not finish")
		}
	}()

	slices, ok := params[InNameFlattenSlice]
	if !ok {
		return "", irr.Error("input param %s is not set", InNameFlattenSlice)
	}

	if !typer.IsSlice(slices) {
		return "", irr.Error("input param %s is not slice", InNameFlattenSlice)
	}

	sliceFlatten := typer.FlattenNestedSlices(slices, 2)
	wlog.ByCtx(ctx, "merge_slice").Infof("merge 2d slice result,\n- form: %v - to: %v", len(slices.([]any)), len(sliceFlatten))
	finished, err = signal(ctx, OutNameFlattenSlice, sliceFlatten)
	return "success, slice merged", nil
}

func (n *WFFlatten2DSlice) Name() string {
	return "flatten_2d_slice"
}

func (n *WFFlatten2DSlice) InNames() []string {
	return []string{InNameFlattenSlice}
}

func (n *WFFlatten2DSlice) OutNames() []string {
	return []string{OutNameFlattenSlice}
}
