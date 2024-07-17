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

// Use
//
// # Connector 语法说明
//
// 标准连接语法：如 A -->|x:y| B
// 简化连接语法：如 A --> B (仅单输入输出)
// 连写连接语法：如 A -->|x:y| B --> Z --> T --> ...
// Prefab 节点语法：如 A -- p|x:y| --> B 和 A -- p --> B
//
// ## 链接语法
// A -->|x:y| B                         	// 将 A 的 x 输出 connect 到 B 的 y 输入，即执行 Connect(A, x, B, y)
// A -->|:y| B                             	// 可以省略上游出参或下游入参，以省略 x 举例，即执行 Connect(A, "__1", B, y)，如果 A 的出参数量不为一，会报错；
// A --> B                              	// 用 A、B 的唯一出参、入参, 进行链接，即执行 Connect(A, "__1", B, "__1")。如果 A 的输出不是一个，或者 B 的输入不是一个，会报错；
// A -->|x:y| B --> Z -->|m:n| T --> ...    // 以上两种链接可以连写，以这个例子来说，执行 Connect(A, x, B, y) 和 Connect(B, "__1", Z, "__1") 和 Connect(Z, m, T, n) ...
//
// ## 参数语法
// 在指定参数名的场合，可以用 __1 来代表第一个参数，是出参还是入参取决于出现的位置
// 比如 A -->|__1:__1| B 表示将 A 的第一个出参传入 B 的第一个入参
//
// ## Prefab 语法，
// 当使用单输入输出的 Prefab 节点时，采用以下语法; 其中 p 指代 Prefab 节点的 key。Prefab 和一般节点的区别是可以重复多次，每次都会调用 clone 方法创建新的 Node 实例
// A -- p|x:y| --> B 					// 将 A 的 x 输出 connect 到 p 的唯一入参，并将 p 的唯一出参 connect 到 B 的 y 输入，即执行 pp = p.CLone(); Connect(A, x, pp, "__1") 和 Connect(pp, "__1", B, "y") . s 的入参和出参必须是一个，否则报错。
// A -- p--> B 							// 如果 A 有唯一出参, B 有唯一入参, C 有唯一的入参和出参, 那自动连接, 他们的参数可以省略. pp = p.CLone(); Connect(A, "__1", pp, "__1") 和 Connect(pp, "__1", B, "__1")
// 举例来说，对于一般节点 C，以下语法会失败，因为 C 作为具体节点，其同一个入参 x 只能接受一个输入
// A --> C|x:y| --> B
// A --> C|x:t| --> T
// 但是对于 prefab 节点，可以重复执行，因为每个 p 都会调用 Node 的一个 Clone() 方法，形成新的实例
// A -- p|x:y| --> B
// A -- p|x:t| --> T
//
// 特殊的，prefab 支持 discard 节点，即将输出指向 discard 标记 `__0`
// A -- p --> & 即执行 pp = p.CLone(); Connect(A, "__1", pp, "__1")，然后就退出
// 可以用于实现类型 A -->|content:input| B -- print --> __0 这样的旁路输出效果
//
// 节点名称和注释：每个节点名称后面可能会紧跟一个或两个括号的注释，例如 C[OK]、C[(OK)]、C{{OK}}、C>OK]。在这种情况下，需要提取节点名称 C 和注释 OK。
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
				return irr.Wrap(err, "connect prefab node [in] %s -->|%s:%s| %s", startNode.Name(), ast.StartOut, SingleNodeParamName, prefabNode.Name())
			}

			if ast.EndNode == DiscardNodeParamName { // 如果 prefab 的下游节点是 discard
				if len(prefabNode.OutNames()) != 0 {
					return fmt.Errorf("discard prefab node %s cannot have out param %v", prefabNode.Name(), prefabNode.OutNames())
				}
			} else {
				if endNode == nil {
					return fmt.Errorf("end node enter is discard, but prefab node %s has out param %v", prefabNode.Name(), prefabNode.OutNames())
				}
				if err := Connect(prefabNode, SingleNodeParamName, endNode, ast.EndIn); err != nil {
					return irr.Wrap(err, "connect prefab node [out] %s -->|%s:%s| %s", prefabNode.Name(), SingleNodeParamName, ast.EndIn, endNode.Name())
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
