package tool

import (
	"context"
	"encoding/json"
	"strings"

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

	ToolManager struct {
		tools map[string]ITool
	}
)

var (
	ErrToolNotFound            = irr.Error("tool not found")
	ErrExecFailedInvalidParams = irr.Error("invalid params")
)

func NewToolManager() *ToolManager {
	return &ToolManager{
		tools: make(map[string]ITool),
	}
}

func (tm *ToolManager) GetTool(name string) (ITool, bool) {
	t, ok := tm.tools[name]
	return t, ok
}

func (tm *ToolManager) RegisterTool(t ITool) {
	tm.tools[t.Name()] = t
}

func (tm *ToolManager) Execute(ctx context.Context, name string, paramValues []string) (any, error) {
	log := wlog.ByCtx(ctx, "ToolManager.Execute")

	log.Debugf("=== try %s with params %v", name, paramValues)
	tool, exists := tm.GetTool(name)
	if !exists {
		return nil, ErrToolNotFound
	}

	paramNames := tool.ParamNames()
	if len(paramNames) != len(paramValues) {
		log.Errorf("parameter list mismatch, expect params of %v, got %s", tool.ParamNames(), paramValues)
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
	return tool.Execute(params)
}
