package workflow

import "context"

type NodeDef interface {
	Execute(ctx context.Context, params ParamsTable, signal SignalTarget) (log string, err error)
	Name() string
	InNames() []string
	OutNames() []string
}

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

// NewNodeByDef 用定义对象创建一个工作流节点
func NewNodeByDef(def NodeDef) Node {
	w := &WN{
		name:      def.Name(),
		executor:  def.Execute,
		EdgeGroup: MakeEdgeGroup(def.InNames(), def.OutNames()),
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
