// Copyright 2023 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// --- Pools & Pre-compiled Query ---

var parserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		parser.SetLanguage(python.GetLanguage())
		return parser
	},
}

var cursorPool = sync.Pool{
	New: func() any { return sitter.NewQueryCursor() },
}

var importQuery *sitter.Query

func init() {
	var err error
	queryString := `
		(import_statement) @import
		(import_from_statement) @import_from
		(comment) @comment
		(if_statement) @if_stmt
	`
	importQuery, err = sitter.NewQuery([]byte(queryString), python.GetLanguage())
	if err != nil {
		panic(fmt.Sprintf("failed to compile import query: %v", err))
	}
}

// --- Constants ---

const (
	sitterNodeTypeString              = "string"
	sitterNodeTypeIdentifier          = "identifier"
	sitterNodeTypeDottedName          = "dotted_name"
	sitterNodeTypeAliasedImport       = "aliased_import"
	sitterNodeTypeWildcardImport      = "wildcard_import"
	sitterNodeTypeImportStatement     = "import_statement"
	sitterNodeTypeComparisonOperator  = "comparison_operator"
	sitterNodeTypeImportFromStatement = "import_from_statement"
)

// --- Types ---

type ParserOutput struct {
	FileName string
	Modules  []module
	Comments []comment
	HasMain  bool
}

type FileParser struct {
	code        []byte
	relFilepath string
	output      ParserOutput
}

func NewFileParser() *FileParser {
	return &FileParser{}
}

// --- Parsing Logic ---

func parseCode(code []byte) (*sitter.Node, error) {
	parser := parserPool.Get().(*sitter.Parser)
	defer parserPool.Put(parser)

	tree, err := parser.ParseCtx(context.Background(), nil, code)
	if err != nil {
		return nil, err
	}

	return tree.RootNode(), nil
}

func (p *FileParser) parseMain(node *sitter.Node) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "if_statement" || child.ChildCount() < 2 {
			continue
		}
		cond := child.Child(1)
		if cond == nil || cond.Type() != sitterNodeTypeComparisonOperator || cond.ChildCount() < 3 {
			continue
		}
		if cond.Child(1).Type() != "==" {
			continue
		}
		a, b := cond.Child(0), cond.Child(2)
		if a == nil || b == nil {
			continue
		}
		// convert "'__main__' == __name__" to "__name__ == '__main__'"
		if b.Type() == sitterNodeTypeIdentifier {
			a, b = b, a
		}
		if a.Type() == sitterNodeTypeIdentifier && a.Content(p.code) == "__name__" &&
			b.Type() == sitterNodeTypeString && string(p.code[b.StartByte()+1:b.EndByte()-1]) == "__main__" {
			return true
		}
	}
	return false
}

func (p *FileParser) isTypeCheckingBlock(node *sitter.Node) bool {
	if node.Type() != "if_statement" || node.ChildCount() < 2 {
		return false
	}
	condition := node.Child(1)
	if condition.Type() == sitterNodeTypeIdentifier && condition.Content(p.code) == "TYPE_CHECKING" {
		return true
	}
	if condition.Type() == "attribute" && condition.ChildCount() >= 3 {
		obj, attr := condition.Child(0), condition.Child(2)
		if obj.Type() == sitterNodeTypeIdentifier && obj.Content(p.code) == "typing" &&
			attr.Type() == sitterNodeTypeIdentifier && attr.Content(p.code) == "TYPE_CHECKING" {
			return true
		}
	}
	return false
}

func parseImportStatement(node *sitter.Node, code []byte) (module, bool) {
	switch node.Type() {
	case sitterNodeTypeDottedName:
		return module{
			Name:       node.Content(code),
			LineNumber: node.StartPoint().Row + 1,
		}, true
	case sitterNodeTypeAliasedImport:
		return parseImportStatement(node.Child(0), code)
	case sitterNodeTypeWildcardImport:
		return module{
			Name:       "*",
			LineNumber: node.StartPoint().Row + 1,
		}, true
	}
	return module{}, false
}

func (p *FileParser) parseImportStatements(node *sitter.Node) {
	if node.Type() == sitterNodeTypeImportStatement {
		for j := 1; j < int(node.ChildCount()); j++ {
			m, ok := parseImportStatement(node.Child(j), p.code)
			if !ok {
				continue
			}
			m.Filepath = p.relFilepath
			if strings.HasPrefix(m.Name, ".") {
				continue
			}
			p.output.Modules = append(p.output.Modules, m)
		}
	} else if node.Type() == sitterNodeTypeImportFromStatement {
		if node.ChildCount() < 4 {
			return
		}
		from := node.Child(1).Content(p.code)
		if strings.HasPrefix(from, ".") {
			return
		}
		for j := 3; j < int(node.ChildCount()); j++ {
			m, ok := parseImportStatement(node.Child(j), p.code)
			if !ok {
				continue
			}
			m.Filepath = p.relFilepath
			m.From = from
			m.Name = fmt.Sprintf("%s.%s", from, m.Name)
			p.output.Modules = append(p.output.Modules, m)
		}
	}
}

func (p *FileParser) SetCodeAndFile(code []byte, relPackagePath, filename string) {
	p.code = code
	p.relFilepath = filepath.Join(relPackagePath, filename)
	p.output = ParserOutput{FileName: filename}
}

// Parse uses pre-compiled tree-sitter queries to extract imports and comments
// from the parsed AST, replacing the previous recursive traversal approach.
func (p *FileParser) Parse(ctx context.Context) (*ParserOutput, error) {
	rootNode, err := parseCode(p.code)
	if err != nil {
		return nil, err
	}

	p.output.HasMain = p.parseMain(rootNode)

	cursor := cursorPool.Get().(*sitter.QueryCursor)
	defer cursorPool.Put(cursor)

	cursor.Exec(importQuery, rootNode)

	seenImports := make(map[uint32]bool)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		for _, capture := range match.Captures {
			captureName := importQuery.CaptureNameForId(capture.Index)
			node := capture.Node

			switch captureName {
			case "import", "import_from":
				if seenImports[node.StartByte()] {
					continue
				}
				seenImports[node.StartByte()] = true
				p.parseImportStatements(node)

			case "comment":
				p.output.Comments = append(p.output.Comments, comment(node.Content(p.code)))

			case "if_stmt":
				if p.isTypeCheckingBlock(node) {
					for j := 0; j < int(node.ChildCount()); j++ {
						subChild := node.Child(j)
						if subChild.Type() == "block" {
							for k := 0; k < int(subChild.ChildCount()); k++ {
								stmt := subChild.Child(k)
								if stmt.Type() == sitterNodeTypeImportStatement || stmt.Type() == sitterNodeTypeImportFromStatement {
									if !seenImports[stmt.StartByte()] {
										seenImports[stmt.StartByte()] = true
										p.parseImportStatements(stmt)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return &p.output, nil
}

func (p *FileParser) ParseFile(ctx context.Context, repoRoot, relPackagePath, filename string) (*ParserOutput, error) {
	code, err := os.ReadFile(filepath.Join(repoRoot, relPackagePath, filename))
	if err != nil {
		return nil, err
	}
	p.SetCodeAndFile(code, relPackagePath, filename)
	return p.Parse(ctx)
}
