package workflow

import (
	"context"
	"fmt"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/irr"
)

type (
	Connector struct {
		firstError error
	}

	ConnectPlaceholder = string
)

const (
	DiscardNodeParamName ConnectPlaceholder = "__0"
	SingleNodeParamName  ConnectPlaceholder = "__1"
)

// Connect connects two nodes by setting the downstream and upstream relationships.
func Connect(from Node, outParamName string, to Node, inParamName string) error {
	if inParamName == SingleNodeParamName {
		if len(to.InNames()) == 0 {
			return irr.Error("node %s has no input param", to.Name())
		} else if len(to.InNames()) > 1 {
			return irr.Error("node %s has more than 1 input param", to.Name())
		}
		inParamName = to.InNames()[0]
	}

	if outParamName == SingleNodeParamName {
		if len(from.OutNames()) == 0 {
			return irr.Error("node %s has no input param", to.Name())
		} else if len(from.OutNames()) > 1 {
			return irr.Error("node %s has more than 1 input param", to.Name())
		}
		outParamName = from.OutNames()[0]
	}

	if err := from.InsertDownstream(outParamName, to); err != nil {
		return err
	}
	if err := to.InsertUpstream(from, outParamName, inParamName); err != nil {
		return err
	}

	return nil
}

func (c *Connector) Connect(ctx context.Context, startNode Node, startOut string, endNode Node, endIn string) *Connector {
	if c.firstError != nil {
		return c // skip
	}
	strConn := fmt.Sprintf("connect %s -->|%s:%s| %s", startNode.Name(), startOut, endIn, endNode.Name())
	if err := Connect(startNode, startOut, endNode, endIn); err != nil {
		c.firstError = irr.Wrap(err, strConn)
		return c
	}
	wlog.ByCtx(ctx, "connector.connect").Info(strConn)
	return c
}

func (c *Connector) Error() error {
	return c.firstError
}
