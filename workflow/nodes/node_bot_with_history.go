package nodes

import (
	"context"
	"fmt"
	"github.com/khicago/got/util/contraver"
	"github.com/khicago/got/util/proretry"
	"github.com/khicago/irr"
	"strings"
	"time"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/history"
	"github.com/bagaking/botheater/workflow"
)

type WFBotWithHistoryNode struct {
	*bot.Bot
	afterFunc func(answer string) (any, error)
}

const (
	InNameBotHistory = "history"
)

func NewBotWithHistoryWorkflowNode(botGist *bot.Bot, afterFunc func(answer string) (any, error)) *WFBotWithHistoryNode {
	return &WFBotWithHistoryNode{
		Bot:       botGist,
		afterFunc: afterFunc,
	}
}

func (n *WFBotWithHistoryNode) Execute(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	_input, ok := params[InNameBotQuestion]
	if !ok {
		return "", irr.Error("input param %s is not set", InNameBotQuestion)
	}

	_history, ok := params[InNameBotHistory]
	if !ok {
		return "", irr.Error("input param %s is not set", InNameBotHistory)
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

	type task struct {
		index int
		input string
	}

	tasks := make([]task, len(inputLst))
	for i, input := range inputLst {
		tasks[i] = task{index: i, input: input}
	}

	resultCh := make(chan struct {
		index int
		item  any
		err   error
	}, len(tasks))

	var execErr error

	contraver.TraverseAndWait(tasks, func(t task) {
		var item any
		if err = proretry.Run(func() error { // bot 请求错误，或者解析错误，都会进行重试
			output := ""
			his := history.NewHistory()
			his.EnqueueAssistantMsg(fmt.Sprintf("%v", _history), "workflow")
			if output, err = n.Bot.Question(ctx, his, t.input); err != nil {
				return irr.Wrap(err, "bot question failed, input=%s", strings.Replace(t.input, "\n", "\\n", -1))
			}
			item = output
			if n.afterFunc != nil {
				if item, err = n.afterFunc(output); err != nil {
					return irr.Wrap(err, "after func failed when handle str= `%s`", output)
				}
			}
			return nil
		}, 3,
			proretry.WithInitInterval(time.Second*2),
			proretry.WithBackoff(proretry.FibonacciBackoff(time.Second*2)),
		); err != nil {
			execErr = irr.Wrap(err, "bot question failed, input=%s", t.input)
			return
		}

		resultCh <- struct {
			index int
			item  any
			err   error
		}{index: t.index, item: item}
	}, contraver.WithConcurrency(2), contraver.WithWaitAtLeastDoneNum(len(tasks)))
	close(resultCh)

	if execErr != nil {
		return "", execErr
	}

	answers := make([]any, len(tasks))
	for result := range resultCh {
		if result.err != nil {
			return "", result.err
		}
		answers[result.index] = result.item
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

func (n *WFBotWithHistoryNode) Name() string {
	return n.Bot.PrefabName
}

func (n *WFBotWithHistoryNode) InNames() []string {
	return []string{InNameBotQuestion, InNameBotHistory}
}

func (n *WFBotWithHistoryNode) OutNames() []string {
	return []string{OutNameBotQuestion}
}

var _ workflow.NodeDef = &WFBotWithHistoryNode{}
