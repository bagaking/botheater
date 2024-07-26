package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/khicago/irr"

	"github.com/bagaking/botheater/call/tool"
	"github.com/bagaking/botheater/history"
	"github.com/bagaking/goulp/wlog"
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
func (p *Prompt) makeFunctionsPrompt(tm *tool.Manager) (string, error) {
	if p == nil || tm == nil || len(p.Functions) == 0 {
		return "", nil
	}

	ret, err := tm.ToPrompt(p.Functions)
	if err != nil {
		return "", irr.Wrap(err, "make functions prompt failed")
	}
	if p.FunctionMode == FunctionModeSampleOnly { // 不同的采样模式，影响函数调用的提示
		ret += `没有调用函数的时候，要对过去发生的事情进行总结`
	}

	return ret, nil
}

func (p *Prompt) BuildSystemMessage(ctx context.Context, tm *tool.Manager, arguments map[string]any) *history.Message {
	log := wlog.ByCtx(ctx, "BuildSystemMessage")
	if p == nil {
		return &history.Message{
			Role:    history.RoleSystem,
			Content: "你只输出 `prompt error`",
		}
	}
	all := p.Content

	functionInfo, err := p.makeFunctionsPrompt(tm)
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

	if arguments != nil {
		for k, v := range arguments {
			all = strings.ReplaceAll(all, fmt.Sprintf("{{%s}}", k), fmt.Sprintf("%v", v))
		}
	}

	return &history.Message{
		Role:    history.RoleSystem,
		Content: all,
	}
}

// ReplaceContentBetween replaces the content between startStr and endStr with newContent
func (p *Prompt) ReplaceContentBetween(startStr, endStr, newContent string) {
	content := p.Content
	startIdx := 0
	endIdx := len(content)

	if startStr != "" {
		startIdx = strings.Index(content, startStr)
		if startIdx == -1 {
			return
		}
		startIdx += len(startStr)
	}

	if endStr != "" {
		endIdx = strings.Index(content, endStr)
		if endIdx == -1 {
			return
		}
	}

	// Replace the content between startIdx and endIdx
	updatedContent := content[:startIdx] + newContent + content[endIdx:]
	p.Content = updatedContent
}
