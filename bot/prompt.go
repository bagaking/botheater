package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/call/tool"
	"github.com/bagaking/botheater/history"
)

type (
	FunctionCtx  string
	FunctionMode string

	Prompt struct {
		Content string `yaml:"content,omitempty" json:"content,omitempty" `

		// Functions 放在 Prompt 里，处于对于 Functions 调用是 prompt 的一部分来理解
		// todo: 这个有待考虑，因为实际上一切 Agent 行为都会体现在 Prompt 上，且 Example 可以基于知识库生成，因此这一层或许未必需要
		Functions []string `yaml:"functions,omitempty" json:"functions,omitempty"`

		FunctionCtx  `yaml:"function_ctx,omitempty" json:"function_ctx,omitempty"`
		FunctionMode `yaml:"function_mode,omitempty" json:"function_mode,omitempty"`
	}
)

const (
	FuncTellStart = `# 现在支持了以下 functions (example 中省略了 func_call:: 前缀)
`
	FuncTellTail = `
## Constrains - Functions
- 当且仅当要使用 function 时，回复 func_call::name(params)
- 要调用 function 时，你只说两句话，第一句是判断依据，第二句是就是 func_call::search(\"用户的问题\")  调用，然后就不任何内容
- 如果不需要调用 function, 你的回复一定不要包含这种格式
- 不允许输出空内容，不知道能做什么时说明即可
`

	// FunctionModePrivateOnly 遗忘模式, function 调用过程不会到原始上下文
	FunctionModePrivateOnly FunctionMode = "private"
	// FunctionModeSampleOnly 采样模式, 要求 agent 将 function 调用总结成 sample，只有 sample 会到原始上下文
	FunctionModeSampleOnly FunctionMode = "sample"
	// FunctionModeDump 复制模式, 将这个过程携带在返回中
	FunctionModeDump FunctionMode = "dump"

	FunctionCtxLocal FunctionCtx = "local"
	FunctionCtxAll   FunctionCtx = "all"
)

// 获取函数信息
func (p *Prompt) makeFunctions(tm *tool.Manager) (string, error) {
	if p == nil || tm == nil || tm.Count() == 0 || len(p.Functions) == 0 {
		return "", nil
	}

	info := FuncTellStart
	for i, fnName := range p.Functions {
		t, ok := tm.GetTool(fnName)
		if !ok {
			return "", irr.Error("Error: function %s not found", fnName)
		}
		info += fmt.Sprintf("%d. %s ; usage: %s ;\n  example: %v;\n", i+1, t.Name(), t.Usage(), t.Examples())
	}
	ret := info + FuncTellTail
	if p.FunctionMode == FunctionModeSampleOnly {
		ret += `没有调用函数的时候，要对过去发生的事情进行总结`
	}

	return ret, nil
}

func (p *Prompt) BuildSystemMessage(ctx context.Context, tm *tool.Manager) *history.Message {
	log := wlog.ByCtx(ctx, "BuildSystemMessage")
	if p == nil {
		return &history.Message{
			Role:    history.RoleSystem,
			Content: "你只输出 `prompt error`",
		}
	}
	all := p.Content

	functionInfo, err := p.makeFunctions(tm)
	if err != nil { // todo: 考虑下，当任一 function 没有加载，则都不会加载
		log.WithError(err).Warnf("build functions failed")
	}
	if functionInfo != "" {
		all += "\n\n" + functionInfo
	}

	if !strings.Contains(all, "# Initialization") {
		all += `
# Initialization
	You must follow the Constrains. Then introduce yourself and introduce the Workflow.`
	}

	return &history.Message{
		Role:    history.RoleSystem,
		Content: all,
	}
}
