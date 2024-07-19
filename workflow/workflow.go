package workflow

import (
	"context"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/contraver"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
)

type (
	// Workflow 表示一个工作流
	Workflow struct {
		Name      string
		StartNode Node
		EndNode   Node
		Output    ParamsTable

		fakeN Node
	}
)

var ErrWorkflowIsNotFinish = irr.Error("workflow is not finish")

func New(name string) *Workflow {
	return &Workflow{
		Name:   name,
		Output: nil,
	}
}

func (wf *Workflow) Finished() bool {
	return wf.Output != nil
}

// SetStartNode 设置工作流的起始节点
// 起始节点所有输入都被接受，只需要指定出参，outputParams 表示起始节点的出参
func (wf *Workflow) SetStartNode(outputParams []string) error {
	wf.StartNode = NewNode("__start", func(ctx context.Context, params ParamsTable, signal SignalTarget) (string, error) {
		triggerFinished := false
		for paramName := range params {
			v, ok := params[paramName] // start 节点取出所有的输入值，并触发下游
			if !ok {
				return "", irr.Error("input param %s of start node is not set", paramName)
			}
			fin, err := signal(ctx, paramName, v)
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

	if wf.fakeN == nil {
		wf.fakeN = NewNode("__fake_node", nil, nil, nil)
	}

	for _, paramName := range outputParams { // 按照 initParams 定义参数
		if err := wf.StartNode.InsertUpstream(wf.fakeN, paramName, paramName); err != nil {
			return err
		}
	}
	return nil
}

// SetEndNode 设置工作流的结束节点
// 终止节点所有的结果都会输出，因此只需要指定入参，inputParams 表示结束节点的入参
func (wf *Workflow) SetEndNode(inputParams []string) {
	wf.EndNode = NewNode("__end", func(ctx context.Context, params ParamsTable, signal SignalTarget) (string, error) {
		if params == nil {
			params = make(ParamsTable)
		}
		wlog.ByCtx(ctx).Infof("workflow %s enter end phase, out= %v", wf.Name, params)
		// wf.Output 被设置时，整个 workflow 就结束了
		wf.Output = ParamsTable{}
		for _, key := range inputParams {
			wf.Output[key] = params[key]
		}

		// todo：并没有下游，现在会检查 TargetTable。但可以考虑跳过执行的实现
		//f, err := signal(ctx, "output", params)
		//if err != nil {
		//	return "", err
		//}
		//if f != true {
		//	return "", irr.Error("end node is not finish")
		//}

		return "success", nil
	}, inputParams, []string{"output"})
}

func (wf *Workflow) callStart(ctx context.Context, initParams ParamsTable) error {
	wlog.ByCtx(ctx, "call_start").Infof("start")
	for paramName := range initParams {
		if _, err := wf.StartNode.In(ctx, wf.fakeN, paramName, initParams[paramName]); err != nil {
			return irr.Wrap(err, "inject init param %s to start node failed", paramName)
		}
	}
	return nil
}

// Execute 依次执行工作流中的所有节点
func (wf *Workflow) Execute(ctx context.Context, initParams ParamsTable) (ParamsTable, error) {
	logger := wlog.ByCtx(ctx, "WF.Execute").WithField("workflow", wf.Name)
	if wf.Finished() {
		return nil, irr.Error("cannot execute a finished workflow")
	}

	// 遍历, 检查所有节点是否都已 Set, 检查是否无环，检查从 StartNode 可达全部节点，检查从全部节点可达 EndNode
	// input 是注册制的，出现环时，由于整个还上所有的节点都至少有一个参数无法满足，因此导致 workflow is not finish 错误
	// 如果允许任务会饿死
	if err := wf.Validate(); err != nil {
		return nil, irr.Wrap(err, "workflow validate failed")
	}

	executionList := make([]Node, 0)
	if err := wf.callStart(ctx, initParams); err != nil {
		return nil, irr.Wrap(err, "call start failed")
	}
	executionList = append(executionList, wf.StartNode)

	allExecuted := make([]Node, 0)
	// 如果检查过了 EndNode 可达，正常执行的情况下一定会有 wf.Output
	for !wf.Finished() {
		// 并发执行所有已就绪的节点
		logger.Infof("start execute nodes: %v", typer.SliceMap(executionList, func(n Node) string { return n.String() }))
		executedNodes, err := executeNodesConcurrently(ctx, executionList)
		if err != nil {
			return nil, err
		}
		allExecuted = append(allExecuted, executedNodes...)

		// 执行过的节点必然完成，全部都可以踢掉
		logger.Infof("remove executed nodes: %v", typer.SliceMap(executedNodes, func(n Node) string { return n.String() }))
		executionList, _ = typer.SliceDiff(executedNodes, executionList)

		// 查询所有执行节点的 downstream 中未就绪的节点，并加入执行列表 (注意，这里不再计算 n.IsSet)
		// 这个筛选 IsAllInputReady 没有问题, 因为如果节点未就绪，那么它一定还有上游在 executionList 中
		// 而 IsSet 应该在运行前检查，和每个 node 的 Execute 里检查，而不是在调度层
		downstream := typer.SliceFilter(getAllDownstreamNodes(executedNodes), func(n Node) bool { return n.IsAllInputReady() })
		logger.Infof("find input_ready downstreams: %v", typer.SliceMap(downstream, func(n Node) string { return n.String() }))

		// 计算新增的节点
		toAdd, _ := typer.SliceDiff(executionList, downstream)
		logger.Infof("add input_ready downstreams: %v", typer.SliceMap(toAdd, func(n Node) string { return n.String() }))
		executionList = append(executionList, toAdd...)

		// 如果执行列表为空，说明所有节点都执行完了
		if len(executionList) == 0 {
			break
		}
	}

	// 说明执行列表为空了，结果还没有出来，这种情况是不可能的
	if !wf.Finished() {
		// 如果没有 Output，说明工作流没有结束, 但却没有可执行的节点了
		return nil, irr.Wrap(ErrWorkflowIsNotFinish, "executed= %v", allExecuted)
	}
	return wf.Output, nil
}

func getAllDownstreamNodes(nodes []Node) []Node {
	downstreamNodes := make([]Node, 0)
	seenNodes := make(map[string]struct{})
	for _, node := range nodes {
		for _, downstream := range node.Downstream() {
			for _, n := range downstream {
				if _, seen := seenNodes[n.UniqID()]; !seen {
					downstreamNodes = append(downstreamNodes, n)
					seenNodes[n.UniqID()] = struct{}{}
				}
			}
		}
	}
	return downstreamNodes
}

func executeNodesConcurrently(ctx context.Context, nodes []Node) ([]Node, error) {
	var executedNodes []Node
	var err error

	contraver.TraverseAndWait(nodes, func(n Node) {
		logger := wlog.ByCtx(ctx, n.String())
		log, execErr := n.Execute(ctx)
		if execErr != nil {
			logger.Errorf("node %s execute failed: %v", n, execErr)
			err = execErr
			return
		}
		logger.Infof("node %s execute success, log: %s", n, log)
		executedNodes = append(executedNodes, n)
	}, contraver.WithConcurrency(len(nodes)), contraver.WithWaitAtLeastDoneNum(len(nodes)))

	if err != nil {
		return nil, err
	}

	return executedNodes, nil
}
