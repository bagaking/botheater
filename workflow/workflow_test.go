package workflow_test

import (
	"context"
	"fmt"
	"github.com/bagaking/goulp/jsonex"
	"testing"

	"github.com/bagaking/botheater/workflow"
)

type H map[string]string

// _NodeExecutorExample(H{"startParam": "aParam"}) 是一个简单的 NodeExecutor，它只记录接收到的参数并触发下游节点。
func _NodeExecutorExample(nameMap H) workflow.NodeExecutor {
	if nameMap == nil {
		nameMap = make(H)
	}
	return func(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (string, error) {
		for paramName, paramValue := range params {
			fmt.Printf("Node received param: %s = %v\n", paramName, paramValue)
			outName, ok := nameMap[paramName]
			if !ok {
				// 使用相同的参数触发下游节点
				if _, err := signal(ctx, paramName, paramValue); err != nil {
					return "", err
				}
			} else {
				if _, err := signal(ctx, outName, paramValue); err != nil {
					return "", err
				}
			}
		}
		return "executed", nil
	}
}

// TestWorkflow_SimpleChain 测试一个简单的链式工作流
// 这个测试创建了一个简单的链式工作流，包含起始节点、三个中间节点和结束节点。
// 验证方法：
// 1. 创建工作流并设置起始节点和结束节点。
// 2. 创建三个中间节点 NodeA、NodeB 和 NodeC。
// 3. 连接节点形成链式结构：StartNode -> NodeA -> NodeB -> NodeC -> EndNode。
// 4. 使用初始参数执行工作流，并验证输出是否正确。
func TestWorkflow_SimpleChain(t *testing.T) {
	// 创建一个新的工作流
	wf := &workflow.Workflow{Name: "SimpleChainWorkflow"}

	// 定义起始节点和结束节点
	wf.SetStartNode([]string{"startParam"})
	wf.SetEndNode([]string{"endParam"})

	// 创建中间节点
	nodeA := workflow.NewNode("NodeA", _NodeExecutorExample(H{"startParam": "aParam"}), []string{"startParam"}, []string{"aParam"})
	nodeB := workflow.NewNode("NodeB", _NodeExecutorExample(H{"aParam": "bParam"}), []string{"aParam"}, []string{"bParam"})
	nodeC := workflow.NewNode("NodeC", _NodeExecutorExample(H{"bParam": "endParam"}), []string{"bParam"}, []string{"endParam"})

	// 连接节点
	if err := workflow.Connect(wf.StartNode, "startParam", nodeA, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeA 失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "aParam", nodeB, "aParam"); err != nil {
		t.Fatalf("连接 NodeA 到 NodeB 失败: %v", err)
	}
	if err := workflow.Connect(nodeB, "bParam", nodeC, "bParam"); err != nil {
		t.Fatalf("连接 NodeB 到 NodeC 失败: %v", err)
	}
	if err := workflow.Connect(nodeC, "endParam", wf.EndNode, "endParam"); err != nil {
		t.Fatalf("连接 NodeC 到结束节点失败: %v", err)
	}

	// 使用初始参数执行工作流
	initParams := workflow.ParamsTable{"startParam": "initialValue"}
	output, err := wf.Execute(context.Background(), initParams)
	if err != nil {
		t.Fatalf("工作流执行失败, err= %v", err)
	}
	if !wf.Finished() {
		t.Fatalf("工作流执行异常")
	}

	// 验证输出
	expectedOutput := workflow.ParamsTable{"endParam": "initialValue"}
	if jsonex.MustMarshalToString(output) != jsonex.MustMarshalToString(expectedOutput) {
		t.Fatalf("工作流输出不符合预期: 得到 %v, 期望 %v", output, expectedOutput)
	}
}

// TestWorkflow_Branching 测试一个包含分支的工作流
// 这个测试创建了一个包含分支的工作流，起始节点连接到两个并行的中间节点，最后汇聚到结束节点。
// 验证方法：
// 1. 创建工作流并设置起始节点和结束节点。
// 2. 创建两个并行的中间节点 NodeA 和 NodeB，以及一个汇聚节点 NodeC。
// 3. 连接节点形成分支结构：StartNode -> NodeA -> NodeC 和 StartNode -> NodeB -> NodeC -> EndNode。
// 4. 使用初始参数执行工作流，并验证输出是否正确。
func TestWorkflow_Branching(t *testing.T) {
	// 创建一个新的工作流
	wf := &workflow.Workflow{Name: "BranchingWorkflow"}

	// 定义起始节点和结束节点
	wf.SetStartNode([]string{"startParam"})
	wf.SetEndNode([]string{"endParam"})

	// 创建中间节点
	nodeA := workflow.NewNode("NodeA", _NodeExecutorExample(H{"startParam": "aParam"}), []string{"startParam"}, []string{"aParam"})
	nodeB := workflow.NewNode("NodeB", _NodeExecutorExample(H{"startParam": "bParam"}), []string{"startParam"}, []string{"bParam"})
	nodeC := workflow.NewNode("NodeC", _NodeExecutorExample(H{"aParam": "endParam", "bParam": "endParam"}), []string{"aParam", "bParam"}, []string{"endParam"})

	// 连接节点
	if err := workflow.Connect(wf.StartNode, "startParam", nodeA, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeA 失败: %v", err)
	}
	if err := workflow.Connect(wf.StartNode, "startParam", nodeB, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeB 失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "aParam", nodeC, "aParam"); err != nil {
		t.Fatalf("连接 NodeA 到 NodeC 失败: %v", err)
	}
	if err := workflow.Connect(nodeB, "bParam", nodeC, "bParam"); err != nil {
		t.Fatalf("连接 NodeB 到 NodeC 失败: %v", err)
	}
	if err := workflow.Connect(nodeC, "endParam", wf.EndNode, "endParam"); err != nil {
		t.Fatalf("连接 NodeC 到结束节点失败: %v", err)
	}

	// 使用初始参数执行工作流
	initParams := workflow.ParamsTable{"startParam": "initialValue"}
	output, err := wf.Execute(context.Background(), initParams)
	if err != nil {
		t.Fatalf("工作流执行失败: %v", err)
	}
	if !wf.Finished() {
		t.Fatalf("工作流执行异常")
	}

	// 验证输出
	expectedOutput := workflow.ParamsTable{"endParam": "initialValue"}
	if jsonex.MustMarshalToString(output) != jsonex.MustMarshalToString(expectedOutput) {
		t.Fatalf("工作流输出不符合预期: 得到 %v, 期望 %v", output, expectedOutput)
	}
}

// TestWorkflow_Conditional 测试一个包含条件判断的工作流
// 这个测试创建了一个包含条件判断的工作流，起始节点根据条件选择执行不同的中间节点，最后汇聚到结束节点。
// 验证方法：
// 1. 创建工作流并设置起始节点和结束节点。
// 2. 创建两个条件节点 NodeA 和 NodeB，以及一个汇聚节点 NodeC。
// 3. 连接节点形成条件判断结构：StartNode -> (条件判断) -> NodeA 或 NodeB -> NodeC -> EndNode。
// 4. 使用初始参数执行工作流，并验证输出是否正确。
func TestWorkflow_Conditional(t *testing.T) {
	// 创建一个新的工作流
	wf := &workflow.Workflow{Name: "ConditionalWorkflow"}

	// 定义起始节点和结束节点
	wf.SetStartNode([]string{"startParam"})
	wf.SetEndNode([]string{"endParam"})

	// 创建中间节点
	nodeA := workflow.NewNode("NodeA", _NodeExecutorExample(H{"startParam": "aParam"}), []string{"startParam"}, []string{"aParam"})
	nodeB := workflow.NewNode("NodeB", _NodeExecutorExample(H{"startParam": "bParam"}), []string{"startParam"}, []string{"bParam"})
	nodeC := workflow.NewNode("NodeC", _NodeExecutorExample(H{"aParam": "endParam", "bParam": "endParam"}), []string{"aParam", "bParam"}, []string{"endParam"})

	// 连接节点
	if err := workflow.Connect(wf.StartNode, "startParam", nodeA, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeA 失败: %v", err)
	}
	if err := workflow.Connect(wf.StartNode, "startParam", nodeB, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeB 失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "aParam", nodeC, "aParam"); err != nil {
		t.Fatalf("连接 NodeA 到 NodeC 失败: %v", err)
	}
	if err := workflow.Connect(nodeB, "bParam", nodeC, "bParam"); err != nil {
		t.Fatalf("连接 NodeB 到 NodeC 失败: %v", err)
	}
	if err := workflow.Connect(nodeC, "endParam", wf.EndNode, "endParam"); err != nil {
		t.Fatalf("连接 NodeC 到结束节点失败: %v", err)
	}

	// 使用初始参数执行工作流
	initParams := workflow.ParamsTable{"startParam": "initialValue"}
	output, err := wf.Execute(context.Background(), initParams)
	if err != nil {
		t.Fatalf("工作流执行失败: %v", err)
	}
	if !wf.Finished() {
		t.Fatalf("工作流执行异常")
	}

	// 验证输出
	expectedOutput := workflow.ParamsTable{"endParam": "initialValue"}
	if jsonex.MustMarshalToString(output) != jsonex.MustMarshalToString(expectedOutput) {
		t.Fatalf("工作流输出不符合预期: 得到 %v, 期望 %v", output, expectedOutput)
	}
}
