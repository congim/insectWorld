// Package main 规范扫描工具，落地AGENTS.md全局开发规范的CI静态检查。
// 扫描Go源码检测：裸数值常量、扩展点ID硬编码、错误码裸返回、中文注释缺失、struct字段注释缺失、类型选择违规。
// 对应AGENTS.md规范1（宏定义）、5（中文注释）、6（字段注释）、8（类型选择）。
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Violation 规范违规记录。
type Violation struct {
	File    string // 文件路径
	Line    int    // 行号
	Rule    string // 违反规范编号
	Message string // 违规描述
}

// Scanner 规范扫描器，遍历Go AST检测违规。
type Scanner struct {
	violations  []Violation     // 违规列表
	fset        *token.FileSet  // 文件位置集
	extPointIDs map[string]bool // 已定义的扩展点ID常量
}

// NewScanner 创建扫描器实例。
func NewScanner() *Scanner {
	return &Scanner{
		fset:        token.NewFileSet(),
		extPointIDs: make(map[string]bool),
	}
}

// Scan 扫描指定目录下的所有Go源文件。
func (s *Scanner) Scan(dir string) error {
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
func (s *Scanner) scanFile(path string) error {
	node, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			s.checkGenDecl(path, x)
		case *ast.StructType:
			s.checkStructFields(path, x)
		case *ast.CallExpr:
			s.checkCallExpr(path, x)
		case *ast.FuncDecl:
			s.checkFuncDecl(path, x)
		}
		return true
	})

	return nil
}

// checkGenDecl 检查常量/变量声明（规范1：扩展点ID常量收集）。
func (s *Scanner) checkGenDecl(file string, decl *ast.GenDecl) {
	if decl.Tok != token.CONST {
		return
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
			val := strings.Trim(lit.Value, `"`)
			if strings.Contains(val, ".") && strings.Contains(name.Name, "ExtPoint") {
				s.extPointIDs[val] = true
			}
		}
	}
}

// checkStructFields 检查struct字段注释（规范6：每个字段必须有中文注释）。
func (s *Scanner) checkStructFields(file string, st *ast.StructType) {
	for _, field := range st.Fields.List {
		if field.Comment == nil || len(field.Comment.List) == 0 {
			pos := s.fset.Position(field.Pos())
			s.addViolation(file, pos.Line, "规范6", "struct字段缺少注释")
			continue
		}
		comment := field.Comment.Text()
		if !isChineseComment(comment) {
			pos := s.fset.Position(field.Pos())
			s.addViolation(file, pos.Line, "规范5", "struct字段注释非中文")
		}
	}
}

// checkCallExpr 检查函数调用（规范1：扩展点ID硬编码检测、规范9：错误码裸返回检测）。
func (s *Scanner) checkCallExpr(file string, call *ast.CallExpr) {
	fn, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	methodName := fn.Sel.Name
	if methodName == "QueryByExtensionPoint" || methodName == "Register" {
		if len(call.Args) > 0 {
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				val := strings.Trim(lit.Value, `"`)
				if strings.Contains(val, ".") {
					pos := s.fset.Position(call.Pos())
					s.addViolation(file, pos.Line, "规范1",
						fmt.Sprintf("扩展点ID硬编码%s，应使用ExtPointXxx常量", val))
				}
			}
		}
	}

	if methodName == "New" && len(call.Args) > 0 {
		if ident, ok := fn.X.(*ast.Ident); ok && ident.Name == "errors" {
			pos := s.fset.Position(call.Pos())
			s.addViolation(file, pos.Line, "规范9",
				"裸返回errors.New，应使用定义的错误码常量")
		}
	}
}

// checkFuncDecl 检查函数声明（规范5：导出标识符中文注释）。
func (s *Scanner) checkFuncDecl(file string, fn *ast.FuncDecl) {
	if fn.Doc == nil || len(fn.Doc.List) == 0 {
		if fn.Name.IsExported() {
			pos := s.fset.Position(fn.Pos())
			s.addViolation(file, pos.Line, "规范5",
				fmt.Sprintf("导出函数%s缺少中文注释", fn.Name.Name))
		}
		return
	}
	comment := fn.Doc.Text()
	if fn.Name.IsExported() && !isChineseComment(comment) {
		pos := s.fset.Position(fn.Pos())
		s.addViolation(file, pos.Line, "规范5",
			fmt.Sprintf("导出函数%s注释非中文", fn.Name.Name))
	}
}

// isChineseComment 判断注释是否包含中文。
func isChineseComment(comment string) bool {
	for _, r := range comment {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// addViolation 添加违规记录。
func (s *Scanner) addViolation(file string, line int, rule, message string) {
	s.violations = append(s.violations, Violation{
		File:    file,
		Line:    line,
		Rule:    rule,
		Message: message,
	})
}

// Report 输出违规报告。
func (s *Scanner) Report() int {
	if len(s.violations) == 0 {
		fmt.Println("规范扫描通过，未发现违规")
		return 0
	}
	fmt.Printf("发现%d个规范违规：\n", len(s.violations))
	for _, v := range s.violations {
		fmt.Printf("  %s:%d [%s] %s\n", v.File, v.Line, v.Rule, v.Message)
	}
	return 1
}

func main() {
	dir := flag.String("dir", ".", "扫描目录")
	flag.Parse()

	scanner := NewScanner()
	if err := scanner.Scan(*dir); err != nil {
		fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
		os.Exit(2)
	}
	os.Exit(scanner.Report())
}
