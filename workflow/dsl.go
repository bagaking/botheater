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

// String returns a string representation of the ASTNode.
func (ast *ASTNode) String() string {
	cmStart := typer.IfThen(ast.StartComment == "", "", "["+ast.StartComment+"]")
	cmEnd := typer.IfThen(ast.EndComment == "", "", "["+ast.EndComment+"]")
	outInTest := ast.StartOut + ast.EndIn
	outIn := typer.IfThen(outInTest == "" || outInTest == SingleNodeParamName+SingleNodeParamName, "", "["+ast.StartOut+":"+ast.EndIn+"]")

	return fmt.Sprintf("%s%s --%s%s--> %s[%s]", ast.StartNode, cmStart, ast.PrefabKey, outIn, ast.EndNode, cmEnd)
}

// ParseScript parses the connector script into an AST.
// It processes each line of the script and constructs a linked list of ASTNodes.
func ParseScript(ctx context.Context, script string) (*ASTNode, error) {
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
			wlog.ByCtx(ctx, "connector.parseScript").Warnf("skip not empty line %d, %s", i, err.Error())
			continue
		}
		wlog.ByCtx(ctx, "connector.parseScript").Debugf("successful insert line %d, %s", i, node)

		if root == nil {
			root = node
		}

		if current != nil {
			current.Next = node
		}
		current = node

		// set current to the tail node
		for current.Next != nil {
			current = current.Next
		}
	}

	return root, nil
}

// parseLine parses a single line of the script into an AST node.
// It handles both standard and chained connection syntax.
func parseLine(line string) (*ASTNode, error) {
	// Adjusted regex to handle complex node names and comments
	re := regexp.MustCompile(`(\w+)(?:[\[\(\{]{1,2}([^()\[\]{}]+)[\]\)\}]{1,2})?\s*--\s*(\w*)\|?([^:]*):?([^|]*)\|?\s*-->\s*(\w+)(?:[\[\(\{]{1,2}([^()\[\]{}]+)[\]\)\}]{1,2})?`)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 0 {
		return &ASTNode{
			StartNode:    strings.TrimSpace(matches[1]),
			StartComment: strings.TrimSpace(matches[2]),
			PrefabKey:    strings.TrimSpace(matches[3]),
			StartOut:     typer.Or(strings.TrimSpace(matches[4]), SingleNodeParamName),
			EndIn:        typer.Or(strings.TrimSpace(matches[5]), SingleNodeParamName),
			EndNode:      strings.TrimSpace(matches[6]),
			EndComment:   strings.TrimSpace(matches[7]),
		}, nil
	}

	// Try simplified and chained connection syntax
	re = regexp.MustCompile(`(\w+)(?:[\[\(\{]{1,2}([^()\[\]{}]+)[\]\)\}]{1,2})?\s*-->\s*(\|([^:]*):([^|]*)\|)?\s*(\w+)(?:[\[\(\{]{1,2}([^()\[\]{}]+)[\]\)\}]{1,2})?((\s*-->\s*(\|([^:]*):([^|]*)\|)?\s*(\w+)(?:[\[\(\{]{1,2}([^()\[\]{}]+)[\]\)\}]{1,2})?)*)`)
	matches = re.FindStringSubmatch(line)
	if len(matches) != 0 {
		root := &ASTNode{
			StartNode:    strings.TrimSpace(matches[1]),
			StartComment: strings.TrimSpace(matches[2]),
			StartOut:     typer.Or(strings.TrimSpace(matches[4]), SingleNodeParamName),
			EndIn:        typer.Or(strings.TrimSpace(matches[5]), SingleNodeParamName),
			EndNode:      strings.TrimSpace(matches[6]),
			EndComment:   strings.TrimSpace(matches[7]),
		}
		current := root
		chain := matches[8]
		reChain := regexp.MustCompile(`\s*-->\s*(\|([^:]*):([^|]*)\|)?\s*(\w+)(?:[\[\(\{]{1,2}([^()\[\]{}]+)[\]\)\}]{1,2})?`)
		chainMatches := reChain.FindAllStringSubmatch(chain, -1)
		for _, cm := range chainMatches {
			current.Next = &ASTNode{
				StartNode:    current.EndNode,
				StartComment: current.EndComment,
				StartOut:     typer.Or(strings.TrimSpace(cm[2]), SingleNodeParamName),
				EndIn:        typer.Or(strings.TrimSpace(cm[3]), SingleNodeParamName),
				EndNode:      strings.TrimSpace(cm[4]),
				EndComment:   strings.TrimSpace(cm[5]),
			}
			current = current.Next
		}
		return root, nil
	}

	return nil, fmt.Errorf("invalid syntax: %s", line)
}
