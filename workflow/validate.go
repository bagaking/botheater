package workflow

import (
	"fmt"

	"github.com/khicago/irr"
)

var (
	ErrCycleDetected    = irr.Error("cycle detected in the workflow")
	ErrNodesNotFullySet = irr.Error("not all nodes are fully set")
)

func (wf *Workflow) Validate() error {
	// 检查所有节点是否都已 Set
	if err := wf.checkAllNodesSet(); err != nil {
		return err
	}

	// 检查所有节点的 UniqID 是否唯一
	if err := wf.checkNodeUniqIDUnique(); err != nil {
		return err
	}

	// 检查是否
	// - 无环，
	// - 无悬空入边（入度为 0 的 Source Node)
	// - 无悬空出边（出度为 0 的 Sink Node），但无出参的悬空出边是可以的
	if err := wf.checkWorkflowValidity(); err != nil {
		return err
	}

	return nil
}

func (wf *Workflow) GetAllNodes() []Node {
	nodes := make(map[string]Node)
	var dfs func(node Node)
	dfs = func(node Node) {
		nodes[node.UniqID()] = node
		for _, downstream := range node.Downstream() {
			for _, dNode := range downstream {
				if _, ok := nodes[dNode.UniqID()]; !ok {
					dfs(dNode)
				}
			}
		}
	}
	dfs(wf.StartNode)
	result := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, node)
	}
	return result
}

func (wf *Workflow) checkAllNodesSet() error {
	for _, node := range wf.GetAllNodes() {
		if !node.IsSet() {
			return irr.Wrap(ErrNodesNotFullySet, "node= %s (id=%s)", node.Name(), node.UniqID())
		}
	}
	return nil
}

func (wf *Workflow) checkNodeUniqIDUnique() error {
	uniqIDs := make(map[string]bool)
	for _, node := range wf.GetAllNodes() {
		if uniqIDs[node.UniqID()] {
			return fmt.Errorf("duplicate UniqID found: %s", node.UniqID())
		}
		uniqIDs[node.UniqID()] = true
	}
	return nil
}

func (wf *Workflow) checkWorkflowValidity() error {
	inDegree := make(map[string]int)
	for _, node := range wf.GetAllNodes() {
		inDegree[node.UniqID()] = 0
	}
	for _, node := range wf.GetAllNodes() {
		for _, downstream := range node.Downstream() {
			for _, dNode := range downstream {
				inDegree[dNode.UniqID()]++
			}
		}
	}

	queue := make([]Node, 0)
	for _, node := range wf.GetAllNodes() {
		if inDegree[node.UniqID()] == 0 {
			queue = append(queue, node)
		}
	}

	visitedCount := 0
	reachableFromStart := make(map[string]bool)
	reachableToEnd := make(map[string]bool)
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visitedCount++
		reachableFromStart[node.UniqID()] = true
		reachableToEnd[node.UniqID()] = true

		for _, downstream := range node.Downstream() {
			for _, dNode := range downstream {
				inDegree[dNode.UniqID()]--
				if inDegree[dNode.UniqID()] == 0 {
					queue = append(queue, dNode)
				}
			}
		}
	}

	if visitedCount != len(wf.GetAllNodes()) {
		return irr.Wrap(ErrCycleDetected, "visited= %v, all= %v", visitedCount, len(wf.GetAllNodes()))
	}

	for _, node := range wf.GetAllNodes() {
		if !reachableFromStart[node.UniqID()] {
			return fmt.Errorf("node %s is not reachable from start node", node.UniqID())
		}
		if !reachableToEnd[node.UniqID()] && len(node.Downstream()) > 0 {
			return fmt.Errorf("end node is not reachable from node %s", node.UniqID())
		}
	}

	return nil
}

func (wf *Workflow) getNodeByUniqID(uniqID string) Node {
	var dfs func(node Node) Node
	dfs = func(node Node) Node {
		if node.UniqID() == uniqID {
			return node
		}
		for _, downstream := range node.Downstream() {
			for _, dNode := range downstream {
				if result := dfs(dNode); result != nil {
					return result
				}
			}
		}
		return nil
	}
	return dfs(wf.StartNode)
}
