package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/khicago/irr"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/strs"

	"github.com/bagaking/botheater/call"
)

var Caller = &call.Caller{
	Prefix: CallPrefix,
	Regex:  regexp.MustCompile(`func_call::(\w+)\((.*?)\)`),
}

const (
	CallPrefix = "func_call::"

	FuncPromptStart = `# 现在支持了以下 functions (example 中省略了 func_call:: 前缀)
`
	FuncPromptTail = `
## Constrains - Functions
- 当且仅当要使用 function 时，回复 func_call::name(params)
- 要调用 function 时，你只说两句话，第一句是判断依据，第二句是就是 func_call::search(\"用户的问题\")  调用，然后就不任何内容
- 如果不需要调用 function, 你的回复一定不要包含这种格式
- 不允许输出空内容，不知道能做什么时说明即可
`
)

type (
	Manager struct {
		tools map[string]ITool
	}
)

func (tm *Manager) ToPrompt(functions []string) (string, error) {
	info := FuncPromptStart
	for i, fnName := range functions {
		t, ok := tm.GetTool(fnName)
		if !ok {
			return "", irr.Error("Error: function %s not found", fnName)
		}
		info += fmt.Sprintf("%d. %s ; usage: %s ;\n  example: %v;\n", i+1, t.Name(), t.Usage(), t.Examples())
	}
	return info + FuncPromptTail, nil
}

func NewToolManager() *Manager {
	return &Manager{
		tools: make(map[string]ITool),
	}
}

func (tm *Manager) GetTool(name string) (ITool, bool) {
	t, ok := tm.tools[name]
	return t, ok
}

func (tm *Manager) Count() int {
	return len(tm.tools)
}

func (tm *Manager) RegisterTool(t ITool) {
	tm.tools[t.Name()] = t
}

func (tm *Manager) Execute(ctx context.Context, name string, paramValues []string) call.Result {
	log := wlog.ByCtx(ctx, "Manager.Execute")
	ret := call.Result{
		FunctionName: name,
		ParamValues:  paramValues,
		Caller:       Caller,
	}

	log.Debugf("=== try %s with params %v", name, paramValues)
	tool, exists := tm.GetTool(name)
	if !exists {
		ret.Error = call.ErrToolNotFound
		return ret
	}

	paramNames := tool.ParamNames()
	ret.ExpectedParamNames = paramNames
	if len(paramNames) != len(paramValues) {
		ret.Error = call.ErrParamsLenNotMet
		return ret
	}

	params := make(map[string]string)
	for i, paramName := range paramNames {
		val := strings.TrimSpace(paramValues[i])
		params[paramName] = val
		if strs.StartsWith(val, "\"") {
			err := json.Unmarshal([]byte(val), &val)
			if err != nil {
				continue
			}
			params[paramName] = val
		}
	}

	log.Debugf("=== call %s with params %v", name, params)

	ret.Response, ret.Error = tool.Execute(params)
	return ret
}
