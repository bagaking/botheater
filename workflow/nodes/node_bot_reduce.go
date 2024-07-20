package nodes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/khicago/got/util/proretry"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/history"
	"github.com/bagaking/botheater/workflow"
)

type WFBotReduce struct {
	*bot.Bot
	afterFunc func(answer string) (any, error)
}

func NewBotReduceWorkflowNode(botGist *bot.Bot, afterFunc func(answer string) (any, error)) *WFBotReduce {
	return &WFBotReduce{
		Bot:       botGist,
		afterFunc: afterFunc,
	}
}

func (n *WFBotReduce) Execute(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	_input, ok := params[InNameBotQuestion]
	if !ok {
		return "", irr.Error("input param %s is not set", InNameBotQuestion)
	}

	inputLst := make([]string, 0)
	switch t := _input.(type) {
	case string:
		inputLst = []string{t}
	case []string:
		inputLst = t
	default:
		return "", irr.Error("input param must be string or []string")
	}
	if len(inputLst) == 0 {
		return "", irr.Error("input param is empty")
	}

	var (
		item   any
		output = ""
	)
	for i, input := range inputLst {
		if err = proretry.Run(func() error { // bot 请求错误，或者解析错误，都会进行重试
			if output, err = n.Bot.Question(ctx, history.NewHistory(), fmt.Sprintf("%v\n\n%v", output, input)); err != nil {
				return irr.Wrap(err, "bot question failed, reduce round %d, input= `%s`", i, strings.Replace(input, "\n", "\\n", -1))
			}
			item = output
			if n.afterFunc != nil && i == len(inputLst)-1 {
				if item, err = n.afterFunc(output); err != nil {
					return irr.Wrap(err, "after func failed when handle str= `%s`", output)
				}
			}
			return nil
		}, 3,
			proretry.WithInitInterval(time.Second*2),
			proretry.WithBackoff(proretry.FibonacciBackoff(time.Second*2)),
		); err != nil {
			return "", irr.Wrap(err, "bot question failed, input= `%s`", strings.Replace(input, "\n", "\\n", -1))
		}
	}

	if finish, err := signal(ctx, OutNameBotQuestion, item); err != nil {
		return "", irr.Wrap(err, "signal failed")
	} else if !finish {
		return "", irr.Error("signal not finished")
	}

	return "success", nil
}

func (n *WFBotReduce) Name() string {
	return n.Bot.PrefabName
}

func (n *WFBotReduce) InNames() []string {
	return []string{InNameBotQuestion}
}

func (n *WFBotReduce) OutNames() []string {
	return []string{OutNameBotQuestion}
}

var _ workflow.NodeDef = &WFBotReduce{}
