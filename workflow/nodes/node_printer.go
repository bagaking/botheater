package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/bagaking/goulp/jsonex"
	"github.com/bagaking/goulp/wlog"

	"github.com/bagaking/botheater/workflow"
)

// WFPrinter 收集中间结果并进行输出
// 这是一种自举的控制流程实现
type WFPrinter struct {
	inNames []string
}

var _ workflow.NodeDef = &WFPrinter{}

func NewPrinterWorkflowNode(inNames ...string) *WFPrinter {
	return &WFPrinter{inNames: inNames}
}

func (n *WFPrinter) Execute(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("\n\n%s's Result\n\n", n.Name()))
	for _, name := range n.inNames {
		sb.WriteString(fmt.Sprintf("==== %s ====", name))
		sb.WriteString("\n")
		if v, ok := params[name]; ok {
			if v == nil {
				sb.WriteString("<nil>")
			} else if str, ok := v.(string); ok {
				sb.WriteString(str)
			} else {
				data, err := jsonex.MarshalIndent(v, "", "  ")
				if err != nil {
					sb.WriteString(fmt.Sprintf("%+v", v))
				} else {
					sb.Write(data)
				}
			}
		} else {
			sb.WriteString("<undefined>")
		}
		sb.WriteString("\n\n")
	}

	result := sb.String()
	wlog.ByCtx(ctx, "printer").Info(result)

	return "success", nil
}

func (n *WFPrinter) Name() string {
	return fmt.Sprintf("printer(%s)", strings.Join(n.inNames, ","))
}

func (n *WFPrinter) InNames() []string {
	return n.inNames
}

func (n *WFPrinter) OutNames() []string {
	return []string{}
}
