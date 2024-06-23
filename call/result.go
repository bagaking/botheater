package call

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bagaking/goulp/jsonex"

	"github.com/khicago/irr"
)

type (
	Result struct {
		*Caller
		FunctionName       string   `json:"function_name,omitempty"`
		ParamValues        []string `json:"param_values,omitempty"`
		ExpectedParamNames []string `json:"expected_param_names,omitempty"`
		Response           any      `json:"response,omitempty"`
		Error              error    `json:"error,omitempty"`
	}
)

var (
	ErrHasNoFunctionCall       = irr.Error("has no function call")
	ErrToolNotFound            = irr.Error("tool not found")
	ErrParamsLenNotMet         = irr.Error("params length not met")
	ErrExecFailedInvalidParams = irr.Error("invalid params")
)

func (result *Result) ToPrompt() string {
	ct := result.Caller
	if result.Error != nil {
		if errors.Is(result.Error, ErrHasNoFunctionCall) {
			return fmt.Sprintf("对话中没有找到 " + ct.Prefix + ", 因此没有进行调用")
		}
		if errors.Is(result.Error, ErrToolNotFound) {
			return fmt.Sprintf(ct.Prefix+"%s(%s) 调用错误!\n因为没有找到名字是 %s 的调用，请检查输入是否正确.", result.FunctionName, strings.Join(result.ParamValues, ","), result.FunctionName)
		}
		if errors.Is(result.Error, ErrParamsLenNotMet) {
			return fmt.Sprintf(ct.Prefix+"%s(%s) 调用错误!\n调用 %s 的参数应该是 %s，请检查输入是否正确.", result.FunctionName, strings.Join(result.ParamValues, ","), result.FunctionName, strings.Join(result.ExpectedParamNames, ","))
		}
		return fmt.Sprintf(ct.Prefix+"%s(%s) 调用错误!\n具体错误是: %v", result.FunctionName, strings.Join(result.ParamValues, ","), result.Error)
	}
	strResp := ""
	if str, ok := result.Response.(string); ok {
		strResp = str
	} else {
		strResp = jsonex.MustMarshalToString(result.Response)
	}

	return fmt.Sprintf(ct.Prefix+"%s(%s) 调用成功!\n结果为: %s", result.FunctionName, strings.Join(result.ParamValues, ","), strResp)
}
