package workflow

import (
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		line     string
		expected *ASTNode
		err      bool
	}{
		{
			line: "A --> B",
			expected: &ASTNode{
				StartNode: "A",
				StartOut:  SingleNodeParamName,
				EndIn:     SingleNodeParamName,
				EndNode:   "B",
			},
			err: false,
		},
		{
			line: "A[Start] -->|x:y| B[End]",
			expected: &ASTNode{
				StartNode:    "A",
				StartComment: "Start",
				StartOut:     "x",
				EndIn:        "y",
				EndNode:      "B",
				EndComment:   "End",
			},
			err: false,
		},
		{
			line: "A -- p|x:y| --> B",
			expected: &ASTNode{
				StartNode: "A",
				PrefabKey: "p",
				StartOut:  "x",
				EndIn:     "y",
				EndNode:   "B",
			},
			err: false,
		},
		{
			line: "A -- p --> B[(xx)]",
			expected: &ASTNode{
				StartNode:  "A",
				PrefabKey:  "p",
				StartOut:   SingleNodeParamName,
				EndIn:      SingleNodeParamName,
				EndNode:    "B",
				EndComment: "xx",
			},
			err: false,
		},
		{
			line: "A -- p --> B",
			expected: &ASTNode{
				StartNode: "A",
				StartOut:  SingleNodeParamName,
				EndIn:     SingleNodeParamName,
				EndNode:   "B",
				PrefabKey: "p",
			},
			err: false,
		},
		{
			line: "A -- p -->|:c| B",
			err:  true,
		},
		{
			line: "A -. p .-> B",
			expected: &ASTNode{
				StartNode: "A",
				StartOut:  SingleNodeParamName,
				EndIn:     SingleNodeParamName,
				EndNode:   "B",
				PrefabKey: "p",
			},
			err: false,
		},
		{
			line: "A --> B[x:x] -- C --> D",
			expected: &ASTNode{
				StartNode:  "A",
				StartOut:   SingleNodeParamName,
				EndIn:      SingleNodeParamName,
				EndNode:    "B",
				EndComment: "x:x",
				Next: &ASTNode{
					StartNode:    "B",
					StartComment: "x:x",
					StartOut:     SingleNodeParamName,
					EndIn:        SingleNodeParamName,
					EndNode:      "D",
					PrefabKey:    "C",
				},
			},
		},
		{
			line: "A -- B --> C[x:x] --> D",
			expected: &ASTNode{
				StartNode:  "A",
				StartOut:   SingleNodeParamName,
				EndIn:      SingleNodeParamName,
				EndNode:    "C",
				EndComment: "x:x",
				PrefabKey:  "B",
				Next: &ASTNode{
					StartNode:    "C",
					StartComment: "x:x",
					StartOut:     SingleNodeParamName,
					EndIn:        SingleNodeParamName,
					EndNode:      "D",
				},
			},
		},
		{
			line: "A -->|:y| B",
			expected: &ASTNode{
				StartNode: "A",
				StartOut:  SingleNodeParamName,
				EndIn:     "y",
				EndNode:   "B",
			},
			err: false,
		},
		{
			line: "A -->|x:| B{[x]}",
			expected: &ASTNode{
				StartNode:  "A",
				StartOut:   "x",
				EndIn:      SingleNodeParamName,
				EndNode:    "B",
				EndComment: "x",
			},
			err: false,
		},
		{
			line: "A -->|x:y| B --> C -->|m:n| T",
			expected: &ASTNode{
				StartNode: "A",
				StartOut:  "x",
				EndIn:     "y",
				EndNode:   "B",
				Next: &ASTNode{
					StartNode: "B",
					StartOut:  SingleNodeParamName,
					EndIn:     SingleNodeParamName,
					EndNode:   "C",
					Next: &ASTNode{
						StartNode: "C",
						StartOut:  "m",
						EndIn:     "n",
						EndNode:   "T",
					},
				},
			},
			err: false,
		},
		{
			line: "A --> B([x:xx]) -->|:details| C((C:cc))",
			expected: &ASTNode{
				StartNode:  "A",
				StartOut:   SingleNodeParamName,
				EndIn:      SingleNodeParamName,
				EndNode:    "B",
				EndComment: "x:xx",
				Next: &ASTNode{
					StartNode:    "B",
					StartComment: "x:xx",
					StartOut:     SingleNodeParamName,
					EndIn:        "details",
					EndNode:      "C",
					EndComment:   "C:cc",
				},
			},
			err: false,
		},
		{
			line: "A --> B[x:x] --> C[t:t]",
			expected: &ASTNode{
				StartNode:  "A",
				StartOut:   SingleNodeParamName,
				EndIn:      SingleNodeParamName,
				EndNode:    "B",
				EndComment: "x:x",
				Next: &ASTNode{
					StartNode:    "B",
					StartComment: "x:x",
					StartOut:     SingleNodeParamName,
					EndIn:        SingleNodeParamName,
					EndNode:      "C",
					EndComment:   "t:t",
				},
			},
			err: false,
		},
		{
			line: "A -->|x:y| B([x:xx]) --> C[ttt] -->|m:n| T",
			expected: &ASTNode{
				StartNode:  "A",
				StartOut:   "x",
				EndIn:      "y",
				EndNode:    "B",
				EndComment: "x:xx",
				Next: &ASTNode{
					StartNode:    "B",
					StartComment: "x:xx",
					StartOut:     SingleNodeParamName,
					EndIn:        SingleNodeParamName,
					EndNode:      "C",
					EndComment:   "ttt",
					Next: &ASTNode{
						StartNode:    "C",
						StartComment: "ttt",
						StartOut:     "m",
						EndIn:        "n",
						EndNode:      "T",
					},
				},
			},
			err: false,
		},
		{
			line:     "invalid syntax",
			expected: nil,
			err:      true,
		},
	}

	for _, test := range tests {
		result, err := parseLine(test.line)
		if (err != nil) != test.err {
			t.Errorf("parseLine(%q) error = %+v, expected error = %v", test.line, err, test.err)
			continue
		}
		if !test.err && !compareASTNodes(result, test.expected) {
			t.Errorf("parseLine(%q) not eq, result = %+v, expected = %v", test.line, result.PrintChain(), test.expected.PrintChain())
		}
	}
}

func compareASTNodes(a, b *ASTNode) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.StartNode == b.StartNode &&
		a.StartOut == b.StartOut &&
		a.EndIn == b.EndIn &&
		a.EndNode == b.EndNode &&
		a.StartComment == b.StartComment &&
		a.EndComment == b.EndComment &&
		compareASTNodes(a.Next, b.Next)
}
