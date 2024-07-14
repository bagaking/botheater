package nodes

import (
	"context"

	"github.com/bagaking/botheater/workflow"
	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
)

type WFFlatten2DSlice struct{}

const (
	InNameMergeSlice  = "slices"
	OutNameMergeSlice = "slice"
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

	slices, ok := params[InNameMergeSlice]
	if !ok {
		return "", irr.Error("input param %s is not set", InNameMergeSlice)
	}

	if !typer.IsSlice(slices) {
		return "", irr.Error("input param %s is not slice", InNameMergeSlice)
	}

	var sliceMerged any
	if !typer.Is2DSlice(slices) {
		sliceMerged = slices
	} else if sliceMerged, err = typer.Flatten2DSlice(slices); err != nil {
		return "", irr.Wrap(err, "merge slice failed")
	}

	wlog.ByCtx(ctx, "merge_slice").Infof("merge 2d slice result: %v", sliceMerged)
	finished, err = signal(ctx, OutNameMergeSlice, sliceMerged)
	return "success, slice merged", nil
}

func (n *WFFlatten2DSlice) Name() string {
	return "flatten_2d_slice"
}

func (n *WFFlatten2DSlice) InNames() []string {
	return []string{InNameMergeSlice}
}

func (n *WFFlatten2DSlice) OutNames() []string {
	return []string{OutNameMergeSlice}
}
