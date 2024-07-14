package nodes

import (
	"context"

	"github.com/khicago/got/util/contraver"

	"github.com/khicago/irr"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/history"
	"github.com/bagaking/botheater/workflow"
)

type WFBotNode struct {
	*bot.Bot
	afterFunc func(answer string) (any, error)
}

const (
	InNameBotQuestion  = "question"
	OutNameBotQuestion = "answer"
)

func NewBotWorkflowNode(botGist *bot.Bot, afterFunc func(answer string) (any, error)) *WFBotNode {
	return &WFBotNode{
		Bot:       botGist,
		afterFunc: afterFunc,
	}
}

func (n *WFBotNode) Execute(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	_input, ok := params[InNameBotQuestion]
	if !ok {
		return "", irr.Error("input param is not set")
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

	answers := make([]any, 0, len(inputLst))
	var execErr error

	// 使用 TraverseAndWait 并发执行
	contraver.TraverseAndWait(inputLst, func(input string) {
		output, err := n.Bot.Question(ctx, history.NewHistory(), input)
		if err != nil {
			execErr = irr.Wrap(err, "bot question failed, input=%s", input)
			return
		}
		var item any = output
		if n.afterFunc != nil {
			item, err = n.afterFunc(output)
			if err != nil {
				execErr = irr.Wrap(err, "after func failed when handle str= %s", output)
				return
			}
		}
		answers = append(answers, item)
	}, contraver.WithConcurrency(5)) // 假设并发数为10，可以根据需要调整

	if execErr != nil {
		return "", execErr
	}

	var output any = answers
	if len(answers) == 1 {
		output = answers[0]
	}

	if finish, err := signal(ctx, OutNameBotQuestion, output); err != nil {
		return "", irr.Wrap(err, "signal failed")
	} else if !finish {
		return "", irr.Error("signal not finished")
	}

	return "success", nil
}

func (n *WFBotNode) Name() string {
	return n.Bot.PrefabName
}

func (n *WFBotNode) InNames() []string {
	return []string{InNameBotQuestion}
}

func (n *WFBotNode) OutNames() []string {
	return []string{OutNameBotQuestion}
}

var _ workflow.NodeDef = &WFBotNode{}
