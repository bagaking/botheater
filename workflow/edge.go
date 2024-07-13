package workflow

import (
	"context"

	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
)

type (
	EdgeGroup struct {
		ConditionTable
		TargetTable

		// assertion
		inputParamNames, outputParamNames []string

		// params map
		nameMap map[Node]map[string]string

		// ready
		conditionReady []string
		targetFinish   map[string]int
	}

	conditionNIL   struct{}
	ConditionTable map[string]any
	TargetTable    map[string][]Node
)

var NILCondition = &conditionNIL{}

func MakeEdgeGroup(inputParamNames, outputParamNames []string) EdgeGroup {
	return EdgeGroup{
		ConditionTable: make(ConditionTable),
		TargetTable:    make(TargetTable),

		inputParamNames:  inputParamNames,
		outputParamNames: inputParamNames,

		nameMap:        make(map[Node]map[string]string),
		conditionReady: make([]string, 0),
		targetFinish:   make(map[string]int),
	}
}

func (e *EdgeGroup) ConditionUnmetCount() int {
	return len(e.ConditionTable) - len(e.conditionReady)
}

func (e *EdgeGroup) TargetUnmetCount() int {
	count := 0
	for targetParam := range e.TargetTable {
		fCount := e.targetFinish[targetParam]
		targetCount := len(e.TargetTable[targetParam])
		count = targetCount - fCount
	}
	return count
}

func (e *EdgeGroup) In(ctx context.Context, upstream Node, paramOutName string, data any) (ready bool, err error) {
	nodeTable, ok := e.nameMap[upstream]
	if !ok {
		return false, irr.Error("upstream %s is not found", upstream)
	}

	paramName := ""
	if paramName, ok = nodeTable[paramOutName]; !ok {
		return false, irr.Error("upstream %s is not found", paramOutName)
	}

	v, ok := e.ConditionTable[paramName]
	if ok {
		return false, irr.Error("input param %s are already set (to %v)", ok, v)
	}
	e.ConditionTable[paramName] = data
	e.conditionReady = append(e.conditionReady, paramName)
	return e.ConditionUnmetCount() == 0, nil
}

func (e *EdgeGroup) TriggerAllDownstream(ctx context.Context, upstream Node, paramOutName string, data any) (finish bool, err error) {
	targets, ok := e.TargetTable[paramOutName]
	if !ok {
		return false, irr.Error("targets %s are not found", paramOutName)
	}
	fCount := e.targetFinish[paramOutName]
	if fCount > len(targets) {
		return false, irr.Error("targets %s are already finish", paramOutName)
	}

	for i := fCount; i < len(targets); i++ {
		node := targets[i]
		if _, err = node.In(ctx, upstream, paramOutName, data); err != nil {
			return false, err
		}
		e.targetFinish[paramOutName]++
	}

	return e.TargetUnmetCount() == 0, nil
}

// IsSet 所有上游都已设置
func (e *EdgeGroup) IsSet() bool {
	// 如果没有配置参数校验，那么认为上游无需设置
	if e.inputParamNames == nil {
		return true
	}
	return len(e.ConditionTable) == len(e.inputParamNames)
}

func (e *EdgeGroup) InsertUpstream(upstream Node, paramOutName string, paramInName string) error {
	if typer.IsNil(upstream) {
		return irr.Error("cannot name nil upstream")
	}
	if e.inputParamNames != nil && typer.SliceContains(e.inputParamNames, paramInName) {
		return irr.Error("unsupported input param %s", paramInName)
	}
	if _, ok := e.nameMap[upstream]; !ok {
		e.nameMap[upstream] = make(map[string]string)
	}
	e.nameMap[upstream][paramOutName] = paramInName

	if _, ok := e.ConditionTable[paramInName]; ok {
		return irr.Error("input param %s are already config", ok)
	}
	e.ConditionTable[paramInName] = NILCondition
	return nil
}

// InsertDownstream 注册一个触发下游
func (e *EdgeGroup) InsertDownstream(paramOutName string, downstreamNode Node) error {
	if typer.IsNil(downstreamNode) {
		return irr.Error("cannot insert nil downstream")
	}
	if e.outputParamNames != nil && typer.SliceContains(e.outputParamNames, paramOutName) {
		return irr.Error("unsupported output param %s", paramOutName)
	}
	if lst, ok := e.TargetTable[paramOutName]; !ok {
		e.TargetTable[paramOutName] = []Node{downstreamNode}
	} else {
		e.TargetTable[paramOutName] = append(lst, downstreamNode)
	}
	return nil
}
