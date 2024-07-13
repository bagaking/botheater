package workflow

import (
	"context"

	"github.com/khicago/irr"
)

type (

	// Node 是一个接口，表示工作流中的一个节点
	Node interface {
		Name() string

		UpStream() ConditionTable
		DownStream() TargetTable

		InsertUpstream(upstream Node, paramOutName string, paramInName string) error
		InsertDownstream(paramOutName string, downstreamNode Node) error

		In(ctx context.Context, upstream Node, paramOutName string, data any) (ready bool, err error) // 所有的 input 收集到以后，进行 Execute
		Out(ctx context.Context, paramName string, data any) (ready bool, err error)                  // 所有的 input 收集到以后，进行 Execute

		IsSet() bool
		IsAllInputReady() bool
		IsFinished() bool

		Execute(ctx context.Context) (log string, err error)
	}

	// SignalTarget 触发一次下游，finish 表示所有的下游都触发完成了
	SignalTarget func(ctx context.Context, paramName string, data any) (finish bool, err error)
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

func Connect(from Node, outParamName string, to Node, inParamName string) error {
	if err := from.InsertDownstream(outParamName, to); err != nil {
		return err
	}

	if err := to.InsertUpstream(from, outParamName, inParamName); err != nil {
		return err
	}

	return nil
}

func (w *WN) Name() string {
	return w.name
}

func (w *WN) UpStream() ConditionTable {
	return w.ConditionTable
}

func (w *WN) DownStream() TargetTable {
	return w.TargetTable
}

func (w *WN) IsSet() bool {
	return w.EdgeGroup.IsSet()
}

func (w *WN) IsAllInputReady() bool {
	return w.EdgeGroup.ConditionUnmetCount() == 0
}

func (w *WN) IsFinished() bool {
	return w.EdgeGroup.TargetUnmetCount() == 0
}

func (w *WN) Out(ctx context.Context, paramOutName string, data any) (ready bool, err error) {
	return w.EdgeGroup.TriggerAllDownstream(ctx, w, paramOutName, data)
}

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
