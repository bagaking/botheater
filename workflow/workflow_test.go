package workflow_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bagaking/botheater/workflow"
	"github.com/bagaking/goulp/jsonex"
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
				outs := strings.Split(outName, ",")
				for _, outName = range outs {
					if _, err := signal(ctx, strings.TrimSpace(outName), paramValue); err != nil {
						return "", err
					}
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
	_ = wf.SetStartNode([]string{"startParam"})
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

// TestWorkflow_Loop 测试一个包含循环的工作流
// 这个测试创建了一个包含循环的工作流，起始节点连接到一个中间节点，中间节点再连接回起始节点，形成循环。
// 验证方法：
// 1. 创建工作流并设置起始节点和结束节点。
// 2. 创建中间节点 NodeA, NodeB。
// 3. 连接节点形成循环结构：StartNode -> NodeA -> NodeB -> NodeA。
// 4. 使用初始参数执行工作流，并验证输出是否正确。
func TestWorkflow_Loop(t *testing.T) {
	// 创建一个新的工作流
	wf := &workflow.Workflow{Name: "LoopWorkflow"}

	// 定义起始节点和结束节点
	wf.SetStartNode([]string{"startParam"})
	wf.SetEndNode([]string{"endParam"})

	// 创建中间节点
	nodeA := workflow.NewNode("NodeA", _NodeExecutorExample(H{"startParam": "aParam"}), []string{"startParam", "xParam"}, []string{"aParam"})
	nodeB := workflow.NewNode("NodeB", _NodeExecutorExample(H{"aParam": "xParam,endParam"}), []string{"aParam"}, []string{"endParam", "xParam"})

	// 连接节点形成循环
	if err := workflow.Connect(wf.StartNode, "startParam", nodeA, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeA 失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "aParam", nodeB, "aParam"); err != nil {
		t.Fatalf("连接 NodeA 到 NodeB 失败: %v", err)
	}
	if err := workflow.Connect(nodeB, "endParam", wf.EndNode, "endParam"); err != nil {
		t.Fatalf("连接 NodeB 到终止节点 失败: %v", err)
	}
	if err := workflow.Connect(nodeB, "xParam", nodeA, "xParam"); err != nil {
		t.Fatalf("连接 NodeB 到 NodeA 失败: %v", err)
	}

	// 使用初始参数执行工作流
	initParams := workflow.ParamsTable{"startParam": "initialValue"}
	_, err := wf.Execute(context.Background(), initParams)
	if err == nil {
		t.Fatalf("工作流应该执行失败")
	}
	if !errors.Is(err, workflow.ErrWorkflowIsNotFinish) {
		t.Fatalf("工作流退出原因不符合预期，err= %v, expect= %v", err, workflow.ErrWorkflowIsNotFinish)
	}
}

// TestWorkflow_Parallel 测试一个包含并行执行的工作流
// 这个测试创建了一个包含并行执行的工作流，起始节点连接到两个并行的中间节点，最后汇聚到结束节点。
// 验证方法：
// 1. 创建工作流并设置起始节点和结束节点。
// 2. 创建两个并行的中间节点 NodeA 和 NodeB，以及一个汇聚节点 NodeC。
// 3. 连接节点形成并行结构：StartNode -> NodeA 和 StartNode -> NodeB -> NodeC -> EndNode。
// 4. 使用初始参数执行工作流，并验证输出是否正确。
func TestWorkflow_Parallel(t *testing.T) {
	// 创建一个新的工作流
	wf := &workflow.Workflow{Name: "ParallelWorkflow"}

	// 定义起始节点和结束节点
	wf.SetStartNode([]string{"startParam"})
	wf.SetEndNode([]string{"endParam"})

	// 创建中间节点
	nodeA := workflow.NewNode("NodeA", _NodeExecutorExample(H{"startParam": "aParam"}), []string{"startParam"}, []string{"aParam"})
	nodeB := workflow.NewNode("NodeB", _NodeExecutorExample(H{"startParam": "bParam"}), []string{"startParam"}, []string{"bParam"})
	nodeC := workflow.NewNode("NodeC", _NodeExecutorExample(H{"aParam": "endParam", "bParam": "endParam"}), []string{"aParam", "bParam"}, []string{"endParam"})

	// 连接节点形成并行结构
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

// TestWorkflow_ErrorHandling 测试一个包含错误处理的工作流
// 这个测试创建了一个包含错误处理的工作流，起始节点连接到一个中间节点，中间节点会返回错误，工作流应当正确处理错误。
// 验证方法：
// 1. 创建工作流并设置起始节点和结束节点。
// 2. 创建一个中间节点 NodeA，该节点会返回错误。
// 3. 连接节点形成结构：StartNode -> NodeA -> EndNode。
// 4. 使用初始参数执行工作流，并验证错误是否正确处理。
func TestWorkflow_ErrorHandling(t *testing.T) {
	// 创建一个新的工作流
	wf := &workflow.Workflow{Name: "ErrorHandlingWorkflow"}

	// 定义起始节点和结束节点
	wf.SetStartNode([]string{"startParam"})
	wf.SetEndNode([]string{"endParam"})

	errExpected := fmt.Errorf("intentional error")
	// 创建中间节点
	nodeA := workflow.NewNode("NodeA", func(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (string, error) {
		return "", errExpected
	}, []string{"startParam"}, []string{"aParam"})

	// 连接节点形成结构
	if err := workflow.Connect(wf.StartNode, "startParam", nodeA, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeA 失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "aParam", wf.EndNode, "endParam"); err != nil {
		t.Fatalf("连接 NodeA 到结束节点失败: %v", err)
	}

	// 使用初始参数执行工作流
	initParams := workflow.ParamsTable{"startParam": "initialValue"}
	_, err := wf.Execute(context.Background(), initParams)
	if err == nil {
		t.Fatalf("工作流应当失败，但没有返回错误")
	}

	// 验证错误信息
	if !errors.Is(err, errExpected) {
		t.Fatalf("工作流错误信息不符合预期: 得到 %v, 期望 %v", err.Error(), errExpected)
	}
}

// TestWorkflow_ComplexDependency 测试一个包含复杂依赖关系的工作流
// 这个测试创建了一个包含复杂依赖关系的工作流，起始节点连接到多个中间节点，中间节点之间有复杂的依赖关系，最后汇聚到结束节点。
// 验证方法：
// 1. 创建工作流并设置起始节点和结束节点。
// 2. 创建多个中间节点 NodeA、NodeB、NodeC 和 NodeD。
// 3. 连接节点形成复杂依赖结构：StartNode -> NodeA -> NodeB -> NodeD 和 StartNode -> NodeC -> NodeD -> EndNode。
// 4. 使用初始参数执行工作流，并验证输出是否正确。
func TestWorkflow_ComplexDependency(t *testing.T) {
	// 创建一个新的工作流
	wf := &workflow.Workflow{Name: "ComplexDependencyWorkflow"}

	// 定义起始节点和结束节点
	wf.SetStartNode([]string{"startParam"})
	wf.SetEndNode([]string{"endParam"})

	// 创建中间节点
	nodeA := workflow.NewNode("NodeA", _NodeExecutorExample(H{"startParam": "aParam"}), []string{"startParam"}, []string{"aParam"})
	nodeB := workflow.NewNode("NodeB", _NodeExecutorExample(H{"aParam": "bParam"}), []string{"aParam"}, []string{"bParam"})
	nodeC := workflow.NewNode("NodeC", _NodeExecutorExample(H{"startParam": "cParam"}), []string{"startParam"}, []string{"cParam"})
	nodeD := workflow.NewNode("NodeD", _NodeExecutorExample(H{"bParam": "endParam", "cParam": "endParam"}), []string{"bParam", "cParam"}, []string{"endParam"})

	// 连接节点形成复杂依赖结构
	if err := workflow.Connect(wf.StartNode, "startParam", nodeA, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeA 失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "aParam", nodeB, "aParam"); err != nil {
		t.Fatalf("连接 NodeA 到 NodeB 失败: %v", err)
	}
	if err := workflow.Connect(nodeB, "bParam", nodeD, "bParam"); err != nil {
		t.Fatalf("连接 NodeB 到 NodeD 失败: %v", err)
	}
	if err := workflow.Connect(wf.StartNode, "startParam", nodeC, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeC 失败: %v", err)
	}
	if err := workflow.Connect(nodeC, "cParam", nodeD, "cParam"); err != nil {
		t.Fatalf("连接 NodeC 到 NodeD 失败: %v", err)
	}
	if err := workflow.Connect(nodeD, "endParam", wf.EndNode, "endParam"); err != nil {
		t.Fatalf("连接 NodeD 到结束节点失败: %v", err)
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

// TestWorkflow_MultipleOutputs 测试一个包含多个输出的工作流
// 这个测试创建了一个包含多个输出的工作流，起始节点连接到一个中间节点，中间节点有多个输出，最后汇聚到结束节点。
// 验证方法：
// 1. 创建工作流并设置起始节点和结束节点。
// 2. 创建一个中间节点 NodeA，该节点有多个输出。
// 3. 连接节点形成结构：StartNode -> NodeA -> (多个输出) -> EndNode。
// 4. 使用初始参数执行工作流，并验证输出是否正确。
func TestWorkflow_MultipleOutputs(t *testing.T) {
	// 创建一个新的工作流
	wf := &workflow.Workflow{Name: "MultipleOutputsWorkflow"}

	// 定义起始节点和结束节点
	wf.SetStartNode([]string{"startParam"})
	wf.SetEndNode([]string{"endParam1", "endParam2"})

	// 创建中间节点
	nodeA := workflow.NewNode("NodeA", _NodeExecutorExample(H{"startParam": "aParam1,aParam2"}), []string{"startParam"}, []string{"aParam1", "aParam2"})

	// 连接节点形成结构
	if err := workflow.Connect(wf.StartNode, "startParam", nodeA, "startParam"); err != nil {
		t.Fatalf("连接起始节点到 NodeA 失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "aParam1", wf.EndNode, "endParam1"); err != nil {
		t.Fatalf("连接 NodeA 到结束节点失败: %v", err)
	}
	if err := workflow.Connect(nodeA, "aParam2", wf.EndNode, "endParam2"); err != nil {
		t.Fatalf("连接 NodeA 到结束节点失败: %v", err)
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
	expectedOutput := workflow.ParamsTable{"endParam1": "initialValue", "endParam2": "initialValue"}
	if jsonex.MustMarshalToString(output) != jsonex.MustMarshalToString(expectedOutput) {
		t.Fatalf("工作流输出不符合预期: 得到 %v, 期望 %v", output, expectedOutput)
	}
}
