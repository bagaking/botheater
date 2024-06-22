package bot

import (
	"context"
	"fmt"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/call/tool"
	"github.com/bagaking/botheater/history"
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
- 不允许输出空内容，不知道能做什么时说明即可
`
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
	return info + FuncTellTail, nil
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
	return &history.Message{
		Role:    history.RoleSystem,
		Content: all,
	}
}
