package workflow

import (
	"context"
	"github.com/khicago/got/util/basealphabet"
	"math/rand"
)

type NodeDef interface {
	Execute(ctx context.Context, params ParamsTable, signal SignalTarget) (log string, err error)
	Name() string
	InNames() []string
	OutNames() []string
}

type (
	Options struct {
		eventCallback func(event, log string)
	}

	OptionsFunc func(opts Options) Options
)

func WithEventCallback(fn func(event, log string)) OptionsFunc {
	return func(opts Options) Options {
		opts.eventCallback = fn
		return opts
	}
}

// NewNodeByDef 用定义对象创建一个工作流节点
func NewNodeByDef(def NodeDef, opts ...OptionsFunc) Node {
	wn := newWN(def.Name(), def.Execute, def.InNames(), def.OutNames(), opts...)
	return wn
}

func newWN(name string, executor NodeExecutor, inputParamNames, outputParamNames []string, opts ...OptionsFunc) *WN {
	w := &WN{
		name:      name,
		uniqueID:  basealphabet.EncodeInt64(basealphabet.Base58BitCoin, rand.Int63()),
		executor:  executor,
		EdgeGroup: MakeEdgeGroup(inputParamNames, outputParamNames),
	}
	opt := Options{}
	for _, fn := range opts {
		opt = fn(opt)
	}
	if opt.eventCallback != nil {
		w.eventCallback = opt.eventCallback
	}
	return w
}

// NewNode
// inputParamNames 校验输入参数，如果设置，关联上游时不能指定超出该范围的参数名并且，且如果未关联所有上游时 IsSet 会返回 false
func NewNode(name string, executor NodeExecutor, inputParamNames, outputParamNames []string, opts ...OptionsFunc) Node {
	return newWN(name, executor, inputParamNames, outputParamNames)
}
