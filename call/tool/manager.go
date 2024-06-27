package tool

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/strs"

	"github.com/bagaking/botheater/call"
)

var Caller = &call.Caller{
	Prefix: CallPrefix,
	Regex:  regexp.MustCompile(`func_call::(\w+)\((.*?)\)`),
}

const CallPrefix = "func_call::"

type (
	Manager struct {
		tools map[string]ITool
	}
)

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
