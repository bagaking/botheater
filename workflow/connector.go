package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
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

	// wlog.Common("connector.connect").Infof("connect %s -->|%s:%s| %s", from.Name(), outParamName, inParamName, to.Name())
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
	logger := wlog.ByCtx(ctx, "connector.ast")

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

		if ast.PrefabKey == "" {
			// Handle standard and simplified connection
			if err := Connect(startNode, ast.StartOut, endNode, ast.EndIn); err != nil {
				return irr.Wrap(err, "connect normal node [out] %s", ast)
			}
			logger.Debugf("connect normal node [out] %s -->|%s:%s| %s", startNode, ast.StartOut, ast.EndIn, endNode)
		} else {
			if err := c.connectAstPrefab(ctx, ast.PrefabKey, nodeMap, startNode, endNode, ast.StartOut, ast.EndIn); err != nil {
				return irr.Wrap(err, "connect as prefab node failed, ast_node= %s", ast)
			}
			logger.Debugf("connect prefab between %s -->|%s:%s| %s failed，prefabKey= %s", startNode, ast.StartOut, ast.EndIn, endNode, ast.PrefabKey)
		}

		ast = ast.Next
	}

	return nil
}

func (c *Connector) connectAstPrefab(ctx context.Context, prefabSetting string, nodeMap map[string]Node, startNode, endNode Node, startOutParam, endInParam string) error {
	logger := wlog.ByCtx(ctx, "connector.ast_prefab")

	prefabKeys := typer.SliceFilter(
		typer.SliceMap(strings.Split(prefabSetting, ","), strings.TrimSpace),
		func(s string) bool { return !typer.IsZero(s) },
	)
	logger.Infof("got prefab keys %v form `%v`", prefabKeys, prefabSetting)

	if len(prefabKeys) == 0 {
		return nil
	}

	clonePrefab := func(prefabKey string) (Node, error) {
		if nodeMap[prefabKey] == nil {
			return nil, fmt.Errorf("prefab node not found: `%s`", prefabKey)
		}
		return nodeMap[prefabKey].Clone(), nil
	}

	clone, err := clonePrefab(prefabKeys[0])
	if err != nil {
		return err
	}

	// start => first clone
	if err = Connect(startNode, startOutParam, clone, SingleNodeParamName); err != nil {
		return irr.Wrap(err, "connect start to clone: %s -->|%s:| %s", startNode, startOutParam, clone)
	}

	// clone => clone
	for i := 1; i < len(prefabKeys); i++ {
		newClone, err := clonePrefab(prefabKeys[i])
		if err != nil {
			return err
		}
		if err = Connect(clone, SingleNodeParamName, newClone, SingleNodeParamName); err != nil {
			return irr.Wrap(err, "connect between clone: %s --> %s", clone, newClone)
		}
		clone = newClone
	}

	// last clone => end
	if endNode == nil { // 接地了
		if len(clone.OutNames()) != 0 {
			return irr.Error("discard clone %s cannot have out param %v", clone, clone.OutNames())
		}
		return nil
	}
	if err = Connect(clone, SingleNodeParamName, endNode, endInParam); err != nil {
		return irr.Wrap(err, "connect clone to end: %s -->|:%s| %s", clone, endInParam, endNode)
	}
	return nil
}
