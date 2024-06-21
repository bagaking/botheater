package tool

import (
	"context"
	"regexp"
	"strings"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/irr"
)

const callPrefix = "func_call::"

var FuncCallRegex = regexp.MustCompile(callPrefix + `(\w+)\((.*?)\)`)

func ParseFunctionCall(ctx context.Context, content string) (funcName string, params []string, err error) {
	log := wlog.ByCtx(ctx, "parse_function_call")
	matches := FuncCallRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", nil, irr.Error("invalid function call format")
	}

	funcName = matches[1]
	paramsStr := matches[2]
	if paramsStr == "" {
		return funcName, []string{}, nil
	}
	params = strings.Split(paramsStr, ",")

	log.Infof("find function call: %s ( %v )", funcName, strings.Join(params, ","))
	return funcName, params, nil
}

// HasFunctionCall 判断是否是函数调用
func HasFunctionCall(content string) bool {
	return FuncCallRegex.MatchString(content)
}
