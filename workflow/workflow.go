package workflow

import (
	"context"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/irr"
)

type (
	// Workflow 表示一个工作流
	Workflow struct {
		Name      string
		StartNode Node
		EndNode   Node
		Output    any
	}
)

func (wf *Workflow) SetStartNode(outputParams []string) {
	wf.StartNode = NewNode("__start", func(ctx context.Context, params ConditionTable, signal SignalTarget) (string, error) {
		triggerFinished := false
		for paramName := range params {
			if _, ok := params[paramName]; !ok {
				return "", irr.Error("input param %s is not set", paramName)
			}
			fin, err := signal(ctx, paramName, params)
			if err != nil {
				return "", err
			}
			if fin {
				triggerFinished = true
			}
		}

		if !triggerFinished {
			return "", irr.Error("start node is not finish")
		}
		return "success", nil
	}, nil, outputParams)
}

func (wf *Workflow) SetEndNode(inputParams []string, finish func(output any)) {
	wf.EndNode = NewNode("__end", func(ctx context.Context, params ConditionTable, signal SignalTarget) (string, error) {
		f, err := signal(ctx, "output", params)
		if err != nil {
			return "", err
		}
		if f != true {
			return "", irr.Error("end node is not finish")
		}
		wf.Output = params
		return "success", nil
	}, inputParams, []string{"output"})
}

// Execute 依次执行工作流中的所有节点
func (wf *Workflow) Execute(ctx context.Context, initParams ConditionTable) (any, error) {
	// todo: 遍历, 检查所有节点是否都已 Set, 检查是否无环，检查从 StartNode 可达全部节点，检查从全部节点可达 EndNode

	executionList := make([]Node, 0)

	// 传入初始参数
	for paramName := range initParams {
		// 按照 initParams 定义参数
		if err := wf.StartNode.InsertUpstream(nil, paramName, paramName); err != nil {
			return nil, irr.Wrap(err, "init param %s is not set", paramName)
		}
		// 注入初始参数, 这里的 ready 可以忽略
		if _, err := wf.StartNode.In(ctx, nil, paramName, initParams[paramName]); err != nil {
			return nil, irr.Wrap(err, "init param %s inject failed", paramName)
		}
	}
	executionList = append(executionList, wf.StartNode)

	for wf.Output == nil {
		// 遍历执行列表，执行所有已就绪的节点
		for i := 0; i < len(executionList); i++ {
			node := executionList[i]
			if !node.IsSet() {
				return nil, irr.Error("node %s is not set", node.Name)
			}
			if !node.IsAllInputReady() {
				return nil, irr.Error("node %s is not ready", node.Name)
			}

			// 这个执行完，送事件给下游这活儿就结束了
			log, err := node.Execute(ctx)
			if err != nil {
				return nil, irr.Wrap(err, "node %s execute failed", node.Name)
			}
			wlog.ByCtx(ctx).Infof("node %s execute success, log: %s", node.Name, log)

			for _, nodes := range node.DownStream() {
				for _, n := range nodes {
					if !n.IsSet() {
						return nil, irr.Error("node %s is not ready", n.Name)
					}
					if !n.IsAllInputReady() {
						continue
					}
					executionList = append(executionList, n)
				}
			}

			// 执行完可以从列表删除，因为如果执行完了还没有 finish，说明要不就是有问题，要不就是这个节点有其他入度。
			// 是个 DAG 问题，只要事先检查过这个图无环，且保证 StartNode 可达全部节点，EndNode 全部节点可达，那么就不会有问题
			executionList = append(executionList[:i], executionList[i+1:]...)
			i--
		}

		// 如果执行列表为空，说明所有节点都执行完了
		if len(executionList) == 0 {
			break
		}
	}

	// 如果检查过了 EndNode 可达，那么这里一定会有 Output
	if wf.Output == nil {
		// 如果没有 Output，说明工作流没有结束, 但却没有可执行的节点了
		return nil, irr.Error("workflow is not finish")
	}
	return wf.Output, nil
}
