package workflow

import (
	"context"

	"github.com/khicago/irr"
)

type (
	// Node represents a node in the workflow. Each node can have upstream and downstream nodes,
	// and can process input data and produce output data.
	Node interface {
		// Name returns the name of the node.
		Name() string

		// UpStream returns the condition table of the node, which contains the input parameters.
		UpStream() ConditionTable

		// DownStream returns the target table of the node, which contains the downstream nodes.
		DownStream() TargetTable

		// InsertUpstream inserts an upstream node with the given parameter names.
		InsertUpstream(upstream Node, paramOutName string, paramInName string) error

		// InsertDownstream inserts a downstream node with the given parameter name.
		InsertDownstream(paramOutName string, downstreamNode Node) error

		// In processes the input data from the upstream node.
		// 所有的 input 收集到以后，进行 Execute
		In(ctx context.Context, upstream Node, paramOutName string, data any) (bool, error)

		// Out processes the output data to the downstream nodes.
		Out(ctx context.Context, paramName string, data any) (bool, error)

		// IsSet checks if all upstream parameters are set.
		IsSet() bool

		// IsAllInputReady checks if all input parameters are ready.
		IsAllInputReady() bool

		// IsFinished checks if all downstream nodes have been triggered.
		IsFinished() bool

		// Execute executes the node's logic.
		Execute(ctx context.Context) (string, error)
	}

	// SignalTarget 触发一次下游
	SignalTarget func(ctx context.Context, paramName string, data any) (finish bool, err error)
	// NodeExecutor the handler of executing a node.
	NodeExecutor func(ctx context.Context, params ConditionTable, signal SignalTarget) (log string, err error)

	WN struct {
		EdgeGroup
		name     string
		executor NodeExecutor
	}
)

var _ Node = &WN{}

// NewNode
// inputParamNames 校验输入参数，如果设置，关联上游时不能指定超出该范围的参数名并且，且如果未关联所有上游时 IsSet 会返回 false
func NewNode(name string, executor NodeExecutor, inputParamNames, outputParamNames []string) Node {
	w := &WN{
		name:      name,
		executor:  executor,
		EdgeGroup: MakeEdgeGroup(inputParamNames, outputParamNames),
	}
	return w
}

// Connect connects two nodes by setting the downstream and upstream relationships.
func Connect(from Node, outParamName string, to Node, inParamName string) error {
	if err := from.InsertDownstream(outParamName, to); err != nil {
		return err
	}
	if err := to.InsertUpstream(from, outParamName, inParamName); err != nil {
		return err
	}
	return nil
}

// Name returns the name of the node.
func (w *WN) Name() string {
	return w.name
}

// UpStream returns the condition table of the node.
func (w *WN) UpStream() ConditionTable {
	return w.ConditionTable
}

// DownStream returns the target table of the node.
func (w *WN) DownStream() TargetTable {
	return w.TargetTable
}

// IsSet checks if all upstream parameters are set.
func (w *WN) IsSet() bool {
	return w.EdgeGroup.IsSet()
}

// IsAllInputReady checks if all input parameters are ready.
func (w *WN) IsAllInputReady() bool {
	return w.EdgeGroup.ConditionUnmetCount() == 0
}

// IsFinished checks if all downstream nodes have been triggered.
func (w *WN) IsFinished() bool {
	return w.EdgeGroup.TargetUnmetCount() == 0
}

// Out processes the output data to the downstream nodes.
func (w *WN) Out(ctx context.Context, paramOutName string, data any) (bool, error) {
	return w.EdgeGroup.TriggerAllDownstream(ctx, w, paramOutName, data)
}

// Execute executes the node's logic.
func (w *WN) Execute(ctx context.Context) (string, error) {
	// 合法性检查
	if w.executor == nil {
		return "", irr.Error("node %s has no executor")
	}
	if !w.IsSet() {
		return "", irr.Error("all upstream should be set")
	}

	// 所有的上游参数已经准备就绪
	if !w.IsAllInputReady() {
		return "", irr.Error("node %s is not ready", w.name)
	}
	v, err := w.executor(ctx, w.UpStream(), w.Out)
	if err != nil {
		return "", irr.Wrap(err, "node %s execute failed")
	}
	if !w.IsFinished() {
		return "", irr.Error("node %s internal error, not all target triggered", w.name)
	}
	return v, nil
}
