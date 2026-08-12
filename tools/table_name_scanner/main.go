// Package main 表名扫描器，落地AGENTS.md规范2（数据库表名t_前缀）。
// 扫描infrastructure/persistence下的表名常量定义，校验是否以t_前缀开头。
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 表名正则：t_前缀+蛇形命名
var tableNameRegex = regexp.MustCompile(`^t_[a-z][a-z0-9_]*$`)

// TableNameViolation 表名违规记录。
type TableNameViolation struct {
	File    string // 文件路径
	Line    int    // 行号
	Table   string // 违规表名
	Message string // 违规描述
}

// TableNameScanner 表名扫描器。
type TableNameScanner struct {
	violations []TableNameViolation
	fset       *token.FileSet
}

// NewTableNameScanner 创建表名扫描器实例。
func NewTableNameScanner() *TableNameScanner {
	return &TableNameScanner{
		fset: token.NewFileSet(),
	}
}

// Scan 扫描指定目录下的所有Go源文件。
func (s *TableNameScanner) Scan(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.Contains(path, "vendor") || strings.Contains(path, ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return s.scanFile(path)
	})
}

// scanFile 扫描单个Go源文件。
func (s *TableNameScanner) scanFile(path string) error {
	node, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(node, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				tableName := strings.Trim(lit.Value, `"`)
				if strings.Contains(name.Name, "Table") || strings.Contains(name.Name, "table") {
					if !tableNameRegex.MatchString(tableName) {
						pos := s.fset.Position(lit.Pos())
						s.violations = append(s.violations, TableNameViolation{
							File:    path,
							Line:    pos.Line,
							Table:   tableName,
							Message: fmt.Sprintf("表名%s不符合t_前缀蛇形命名规范", tableName),
						})
					}
				}
			}
		}
		return true
	})
	return nil
}

// Report 输出违规报告。
func (s *TableNameScanner) Report() int {
	if len(s.violations) == 0 {
		fmt.Println("表名扫描通过，所有表名符合t_前缀规范")
		return 0
	}
	fmt.Printf("发现%d个表名违规：\n", len(s.violations))
	for _, v := range s.violations {
		fmt.Printf("  %s:%d 表名=%s %s\n", v.File, v.Line, v.Table, v.Message)
	}
	return 1
}

func main() {
	dir := flag.String("dir", ".", "扫描目录")
	flag.Parse()

	scanner := NewTableNameScanner()
	if err := scanner.Scan(*dir); err != nil {
		fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
		os.Exit(2)
	}
	os.Exit(scanner.Report())
}
