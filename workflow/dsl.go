package workflow

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
)

// ASTNode represents a node in the abstract syntax tree for the connector script.
// Each ASTNode contains information about the start node, end node, parameters, and comments.
// The Next field points to the next node in the chain if there are multiple connections in a single line.
type ASTNode struct {
	StartNode    string   // The name of the start node
	StartOut     string   // The output parameter of the start node
	EndNode      string   // The name of the end node
	EndIn        string   // The input parameter of the end node
	PrefabKey    string   // The key for prefab nodes
	StartComment string   // The comment associated with the start node
	EndComment   string   // The comment associated with the end node
	Next         *ASTNode // The next node in the chain if there are multiple connections
}

const (
	_OpenBrackets_  = `[\[\(\{]{1,2}`                                   // 左括号, matches 0
	_CloseBrackets_ = `[\]\)\}]{1,2}`                                   // 右括号, matches 0
	_Comment_       = `(?:_OpenBrackets_([^()\[\]{}]+)_CloseBrackets_)` // 注释, matches 1: comment
	_Node_          = `(\w+)_Comment_?`                                 // 节点名, matches 2: node name, comment

	_ParamLink_  = `(?:(?:-[.-]*-)|(?:={2,}))[>ox](\|([^:]*):([^|]*)\|)?` // matches 3 : param all, param in, param out
	_PrefabLink_ = `-[-.]\s+(\w*)(?:\|?([^|:]*):?([^|]*)\|?)\s*[-.]->`    // matches 3 : prefab key, prefab in, prefab out

	_ReChained_   = `_Node__ReChainPart_((_ReChainPart_)*)` // matches 11+ = 2 + 8 + 1 + n:
	_ReChainPart_ = `\s*_Link_\s*_Node_`                    // matches 8 = 3 + 3 + 2: link -(1 param, 2 in, 3 out), (4 prefab, 5 in, 6 out), 7 node name, 8 comment
)

func or(subSentences ...string) string {
	orSentence := strings.Join(typer.SliceMap(subSentences, func(s string) string { return fmt.Sprintf(`(?:%s)`, s) }), "|")
	return fmt.Sprintf(`(?:%s)`, orSentence)
}

func compile(str string) *regexp.Regexp {
	str = strings.Replace(str, "_ReChained_", _ReChained_, -1)
	str = strings.Replace(str, "_ReChainPart_", _ReChainPart_, -1)
	str = strings.Replace(str, "_Link_", or(_ParamLink_, _PrefabLink_), -1)
	str = strings.Replace(str, "_Node_", _Node_, -1)
	str = strings.Replace(str, "_Comment_", _Comment_, -1)
	str = strings.Replace(str, "_PrefabLink_", _PrefabLink_, -1)
	str = strings.Replace(str, "_ParamLink_", _ParamLink_, -1)
	str = strings.Replace(str, "_OpenBrackets_", _OpenBrackets_, -1)
	str = strings.Replace(str, "_CloseBrackets_", _CloseBrackets_, -1)

	//wlog.Common().Infof("compile =%v", str)
	re, err := regexp.Compile(str)

	if err != nil {
		panic(err)
	}
	return re
}

var (
	// reChained，用于匹配简化和连写连接语法
	reChained = compile(_ReChained_)
	// 正则表达式，用于匹配连写连接语法中的每个连接部分
	reChainPart = compile(_ReChainPart_)
)

// parseLine parses a single line of the script into an AST node.
// It handles both standard and chained connection syntax.
// parseLine parses a single line of the script into an AST node.
// It handles both standard and chained connection syntax.
func parseLine(line string) (*ASTNode, error) {
	matches := reChained.FindStringSubmatch(line)
	//wlog.Common().Infof("parse line %s ;\nmatches =%v", line, jsonex.MustMarshalToString(matches))
	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid syntax: %s", line)
	}
	// 1 node name, 2 comment, link - (3 prefab, 4 in, 5 out) or (6 param, 7 in, 8 out), 9 node name, 10 comment, 11
	if len(matches) < 11 {
		// todo: 考虑换行指认的情况
		return nil, fmt.Errorf("invalid syntax: %s", line)
	}

	// 检查是否为 Prefab 节点
	root := &ASTNode{
		StartNode:    strings.TrimSpace(matches[1]),
		StartComment: strings.TrimSpace(matches[2]),
		EndNode:      strings.TrimSpace(matches[9]),
		EndComment:   strings.TrimSpace(matches[10]),
	}
	if strings.TrimSpace(matches[3]) != "" {
		root.StartOut = strings.TrimSpace(matches[4])
		root.EndIn = strings.TrimSpace(matches[5])
	}

	if root.PrefabKey = strings.TrimSpace(matches[6]); root.PrefabKey != "" {
		root.StartOut = strings.TrimSpace(matches[7])
		root.EndIn = strings.TrimSpace(matches[8])
	}

	root.StartOut = typer.Or(root.StartOut, SingleNodeParamName)
	root.EndIn = typer.Or(root.EndIn, SingleNodeParamName)

	current := root
	chain := matches[11]

	chainMatches := reChainPart.FindAllStringSubmatch(chain, -1)
	for _, cm := range chainMatches {
		//wlog.Common().Infof("chainIn %s ;;; matches =%v", line, jsonex.MustMarshalToString(cm))
		current.Next = &ASTNode{
			StartNode:    current.EndNode,
			StartComment: current.EndComment,
			EndNode:      strings.TrimSpace(cm[7]),
			EndComment:   strings.TrimSpace(cm[8]),
		}
		if strings.TrimSpace(cm[1]) != "" {
			current.Next.StartOut = strings.TrimSpace(cm[2])
			current.Next.EndIn = strings.TrimSpace(cm[3])
		}

		if current.Next.PrefabKey = strings.TrimSpace(cm[4]); current.Next.PrefabKey != "" {
			current.Next.StartOut = strings.TrimSpace(cm[5])
			current.Next.EndIn = strings.TrimSpace(cm[6])
		}

		current.Next.StartOut = typer.Or(current.Next.StartOut, SingleNodeParamName)
		current.Next.EndIn = typer.Or(current.Next.EndIn, SingleNodeParamName)

		current = current.Next
	}
	return root, nil
}

// ParseScript parses the connector script into an AST.
// It processes each line of the script and constructs a linked list of ASTNodes.
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
// A -- p --> __0 即执行 pp = p.CLone(); Connect(A, "__1", pp, "__1")，然后就退出
// 可以用于实现类型 A -->|content:input| B -- print --> __0 这样的旁路输出效果
//
// 节点名称和注释：每个节点名称后面可能会紧跟一个或两个括号的注释，例如 C[OK]、C[(OK)]、C{{OK}}、C>OK]。在这种情况下，需要提取节点名称 C 和注释 OK。
func ParseScript(ctx context.Context, script string) (*ASTNode, error) {
	logger := wlog.ByCtx(ctx, "connector.parseScript")
	lines := strings.Split(script, "\n")
	var root, current *ASTNode

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "%%") { // Skip comment lines
			continue
		}

		node, err := parseLine(line)
		if err != nil {
			logger.Warnf("skip not empty line %d, line= %s, err= %s", i, line, err.Error())
			continue
		}

		if root == nil {
			root = node
		}

		if current != nil {
			current.Next = node
		}
		current = node
		logger.Debugf("successful insert line %d, %s", i, current)
		// set current to the tail node
		for current.Next != nil {
			current = current.Next
			logger.Debugf("successful insert line .. %d, %s", i, current)
		}
	}

	return root, nil
}

// String returns a string representation of the ASTNode.
func (ast *ASTNode) String() string {
	cmStart := typer.IfThen(ast.StartComment == "", "", "("+ast.StartComment+")")
	cmEnd := typer.IfThen(ast.EndComment == "", "", "("+ast.EndComment+")")
	//outInTest := ast.StartOut + ast.EndIn
	outIn := "|" + ast.StartOut + ":" + ast.EndIn + "|" // always show; typer.IfThen(outInTest == "" || outInTest == SingleNodeParamName+SingleNodeParamName, "", )
	prefabOutInt := typer.IfThen(ast.PrefabKey+outIn == "", "", "- "+ast.PrefabKey+outIn+" -")

	return fmt.Sprintf("%s%s -%s-> %s%s", ast.StartNode, cmStart, prefabOutInt, ast.EndNode, cmEnd)
}

// PrintChain returns a string representation of the ASTNode.
func (ast *ASTNode) PrintChain() string {
	if ast == nil {
		return "<nil>"
	}
	if ast.Next == nil {
		return ast.String()
	}
	return fmt.Sprintf("%s |> %s", ast, ast.Next)
}
