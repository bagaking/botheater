package workflow

import (
	"context"

	"github.com/google/uuid"
)

type NodeDef interface {
	Execute(ctx context.Context, params ParamsTable, signal SignalTarget) (log string, err error)
	Name() string
	InNames() []string
	OutNames() []string
}

// NewNode
// inputParamNames 校验输入参数，如果设置，关联上游时不能指定超出该范围的参数名并且，且如果未关联所有上游时 IsSet 会返回 false
func NewNode(name string, executor NodeExecutor, inputParamNames, outputParamNames []string) Node {
	return newWN(name, executor, inputParamNames, outputParamNames)
}

// NewNodeByDef 用定义对象创建一个工作流节点
func NewNodeByDef(def NodeDef) Node {
	return newWN(def.Name(), def.Execute, def.InNames(), def.OutNames())
}

func newWN(name string, executor NodeExecutor, inputParamNames, outputParamNames []string) *WN {
	w := &WN{
		name:      name,
		uniqueID:  uuid.NewString(),
		executor:  executor,
		EdgeGroup: MakeEdgeGroup(inputParamNames, outputParamNames),
	}
	return w
}
