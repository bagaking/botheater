package bot

import (
	"fmt"

	"github.com/bagaking/botheater/tool"
	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
)

type (
	Prompt struct {
		Content string `yaml:"content,omitempty" json:"content,omitempty" `

		// function names
		Functions []string `yaml:"functions,omitempty" json:"functions,omitempty"`
	}
)

const (
	FuncTellStart = `# 现在支持了以下 functions
`
	FuncTellTail = `
当且仅当要使用 function 时，回复 func_call::name(params)，比如：
func_call::search(\"用户的问题\") 
注意:
- 要调用函数时不要回复除调用函数以外的内容
- 如果不需要调用 function, 你的回复一定不要包含这种格式
`
)

// 获取函数信息
func (p *Prompt) makeFunctions(tm *tool.Manager) string {
	if p == nil || tm == nil {
		return ""
	}

	info := FuncTellStart
	for i, fnName := range p.Functions {
		t, ok := tm.GetTool(fnName)
		if !ok {
			return fmt.Sprintf("Error: function %s not found", fnName)
		}
		info += fmt.Sprintf("%d. %s ; usage: %s ;\n  example: %v;\n", i+1, t.Name(), t.Usage(), t.Examples())
	}
	return info + FuncTellTail
}

func (p *Prompt) BuildSystemMessage(tm *tool.Manager) *api.Message {
	if p == nil {
		return &api.Message{
			Role:    api.ChatRoleSystem,
			Content: "你只输出 `prompt error`",
		}
	}
	all := p.Content
	functionInfo := p.makeFunctions(tm)
	if functionInfo != "" {
		all += "\n\n" + functionInfo
	}
	return &api.Message{
		Role:    api.ChatRoleSystem,
		Content: all,
	}
}
