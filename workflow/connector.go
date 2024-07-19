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
			return irr.Error("node %s has no input param", to.String())
		} else if len(to.InNames()) > 1 {
			return irr.Error("node %s has more than 1 input param", to.String())
		}
		inParamName = to.InNames()[0]
	}

	if outParamName == SingleNodeParamName {
		if len(from.OutNames()) == 0 {
			return irr.Error("node %s has no input param", to.String())
		} else if len(from.OutNames()) > 1 {
			return irr.Error("node %s has more than 1 input param", to.String())
		}
		outParamName = from.OutNames()[0]
	}

	if err := from.InsertDownstream(outParamName, to); err != nil {
		return irr.Wrap(err, "connect %s -->|%s:%s| %s", from.String(), outParamName, inParamName, to.String())
	}

	//wlog.Common("connector.connect").Infof("connect %s -->|%s:%s| %s", from.Name(), outParamName, inParamName, to.Name())
	if err := to.InsertUpstream(from, outParamName, inParamName); err != nil {
		return irr.Wrap(err, "connect %s -->|%s:%s| %s", from.String(), outParamName, inParamName, to.String())
	}

	return nil
}

func (c *Connector) Connect(ctx context.Context, startNode Node, startOut string, endNode Node, endIn string) *Connector {
	if c.firstError != nil {
		return c // skip
	}
	strConn := fmt.Sprintf("connect %s -->|%s:%s| %s", startNode.String(), startOut, endIn, endNode.String())
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

func (c *Connector) Use(ctx context.Context, nodeMap map[string]Node, script string) *Connector {
	ast, err := ParseScript(ctx, script)
	if err != nil {
		c.firstError = err
		return c
	}

	if err = c.connectByAST(ctx, nodeMap, ast); err != nil {
		c.firstError = err
		return c
	}

	return c
}

// connectByAST connects nodes based on the AST.
func (c *Connector) connectByAST(ctx context.Context, nodeMap map[string]Node, ast *ASTNode) error {
	// logger := wlog.ByCtx(ctx, "connector.connectNodes")
	for ast != nil {
		startNode, ok := nodeMap[ast.StartNode]
		if !ok {
			return fmt.Errorf("node not found: %s", ast.StartNode)
		}

		var endNode Node
		if ast.EndNode != DiscardNodeParamName {
			if endNode, ok = nodeMap[ast.EndNode]; !ok {
				return fmt.Errorf("node not found: %s", ast.EndNode)
			}
		}

		if ast.PrefabKey != "" {
			if nodeMap[ast.PrefabKey] == nil {
				return fmt.Errorf("prefab node not found: %s", ast.PrefabKey)
			}
			// Handle prefab node
			prefabNode := nodeMap[ast.PrefabKey].Clone()

			if err := Connect(startNode, ast.StartOut, prefabNode, SingleNodeParamName); err != nil {
				return irr.Wrap(err, "connect prefab node [in] %s -->|%s:%s| %s", startNode, ast.StartOut, SingleNodeParamName, prefabNode)
			}

			if ast.EndNode == DiscardNodeParamName { // 如果 prefab 的下游节点是 discard
				if len(prefabNode.OutNames()) != 0 {
					return fmt.Errorf("discard prefab node %s cannot have out param %v", prefabNode, prefabNode.OutNames())
				}
			} else {
				if endNode == nil {
					return fmt.Errorf("end node enter is discard, but prefab node %s has out param %v", prefabNode, prefabNode.OutNames())
				}
				if err := Connect(prefabNode, SingleNodeParamName, endNode, ast.EndIn); err != nil {
					return irr.Wrap(err, "connect prefab node [out] %s -->|%s:%s| %s", prefabNode, SingleNodeParamName, ast.EndIn, endNode)
				}
			}

		} else {
			// Handle standard and simplified connection
			if err := Connect(startNode, ast.StartOut, endNode, ast.EndIn); err != nil {
				return irr.Wrap(err, "connect normal node [out] %s", ast)
			}
			// logger.Infof("connect normal node [out] %s -->|%s:%s| %s", startNode.Name(), ast.StartOut, ast.EndIn, endNode.Name())
		}

		ast = ast.Next
	}

	return nil
}
