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
		UpStream() ParamsTable

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
	NodeExecutor func(ctx context.Context, params ParamsTable, signal SignalTarget) (log string, err error)

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
func (nThis *WN) Name() string {
	return nThis.name
}

// String
func (nThis *WN) String() string {
	return nThis.name
}

// UpStream returns the condition table of the node.
func (nThis *WN) UpStream() ParamsTable {
	return nThis.ParamsTable
}

// DownStream returns the target table of the node.
func (nThis *WN) DownStream() TargetTable {
	return nThis.TargetTable
}

// IsSet checks if all upstream parameters are set.
func (nThis *WN) IsSet() bool {
	return nThis.EdgeGroup.IsSet()
}

// IsAllInputReady checks if all input parameters are ready.
func (nThis *WN) IsAllInputReady() bool {
	return nThis.EdgeGroup.ConditionUnmetCount() == 0
}

// IsFinished checks if all downstream nodes have been triggered.
func (nThis *WN) IsFinished() bool {
	return nThis.EdgeGroup.TargetUnmetCount() == 0
}

// Out processes the output data to the downstream nodes.
func (nThis *WN) Out(ctx context.Context, paramOutName string, data any) (bool, error) {
	return nThis.EdgeGroup.TriggerAllDownstream(ctx, nThis, paramOutName, data)
}

// Execute executes the node's logic.
func (nThis *WN) Execute(ctx context.Context) (string, error) {
	// 合法性检查
	if nThis.executor == nil {
		return "", irr.Error("node %s has no executor", nThis.name)
	}
	if !nThis.IsSet() {
		return "", irr.Error("all upstream should be set")
	}

	// 所有的上游参数已经准备就绪
	if !nThis.IsAllInputReady() {
		return "", irr.Error("node %s is not ready", nThis.name)
	}
	v, err := nThis.executor(ctx, nThis.UpStream(), nThis.Out)
	if err != nil {
		return "", irr.Wrap(err, "node %v execute failed", nThis.name)
	}
	if !nThis.IsFinished() {
		return "", irr.Error("node %s internal error, not all target triggered", nThis.name)
	}
	return v, nil
}
