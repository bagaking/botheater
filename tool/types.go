package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bagaking/goulp/jsonex"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/strs"
	"github.com/khicago/irr"
)

type (
	ITool interface {
		Execute(params map[string]string) (any, error)
		Name() string
		Usage() string
		Examples() []string
		ParamNames() []string
	}

	Manager struct {
		tools map[string]ITool
	}

	Result struct {
		FunctionName       string
		ParamValues        []string
		ExpectedParamNames []string
		Response           any
		Error              error
	}
)

var (
	ErrHasNoFunctionCall       = irr.Error("has no function call")
	ErrToolNotFound            = irr.Error("tool not found")
	ErrParamsLenNotMet         = irr.Error("params length not met")
	ErrExecFailedInvalidParams = irr.Error("invalid params")
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

func (tm *Manager) RegisterTool(t ITool) {
	tm.tools[t.Name()] = t
}

func (tm *Manager) Execute(ctx context.Context, name string, paramValues []string) Result {
	log := wlog.ByCtx(ctx, "Manager.Execute")
	ret := Result{
		FunctionName: name,
		ParamValues:  paramValues,
	}

	log.Debugf("=== try %s with params %v", name, paramValues)
	tool, exists := tm.GetTool(name)
	if !exists {
		ret.Error = ErrToolNotFound
		return ret
	}

	paramNames := tool.ParamNames()
	ret.ExpectedParamNames = paramNames
	if len(paramNames) != len(paramValues) {
		ret.Error = ErrParamsLenNotMet
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

// ToPrompt 根据 Result 生成对 AI 的修正调用要求
func (result *Result) ToPrompt() string {
	if result.Error != nil {
		if errors.Is(result.Error, ErrHasNoFunctionCall) {
			return fmt.Sprintf("对话中没有找到 " + CallPrefix + ", 因此没有进行 function 调用")
		}
		if errors.Is(result.Error, ErrToolNotFound) {
			return fmt.Sprintf(CallPrefix+"%s(%s) 调用错误!\n因为没有找到名字是 %s 的 function，请检查输入是否正确.", result.FunctionName, strings.Join(result.ParamValues, ","), result.FunctionName)
		}
		if errors.Is(result.Error, ErrParamsLenNotMet) {
			return fmt.Sprintf(CallPrefix+"%s(%s) 调用错误!\nfunction %s 的参数应该是 %s，请检查输入是否正确.", result.FunctionName, strings.Join(result.ParamValues, ","), result.FunctionName, strings.Join(result.ExpectedParamNames, ","))
		}
		return fmt.Sprintf(CallPrefix+"%s(%s) 调用错误!\n具体错误是: %s", result.FunctionName, strings.Join(result.ParamValues, ","), jsonex.MustMarshalToString(result.Error))
	}
	return fmt.Sprintf(CallPrefix+"%s(%s) 调用成功!\n结果为: %s", result.FunctionName, strings.Join(result.ParamValues, ","), jsonex.MustMarshalToString(result.Response))
}
