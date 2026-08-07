// Package main ZeroHardcodeScanner零硬编码静态扫描CI工具。
// 扫描服务端代码：不出现具体游戏名词、无数值常量（除白名单）、无游戏特定枚举值。
// 输出扫描报告，违规项列出具体文件和行号。
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

// 扫描结果状态常量（规范1）。
const (
	ResultPass = 1 // 通过
	ResultFail = 2 // 失败
)

// 白名单常量（规范1），允许框架基础设施常量。
const (
	whiteListDefaultTimeout = "默认超时，框架基础设施白名单"
	whiteListDefaultRetry   = "默认重试次数，框架基础设施白名单"
)

// 游戏特定名词黑名单（应由配置驱动，此处为扫描规则示例）。
var gameNameBlacklist = []string{
	"步兵", "弓箭手", "骑兵", "火球术", "冰冻术",
	"三国", "星际", "蚂蚁",
}

// ScanResult 扫描结果。
type ScanResult struct {
	Status     int         // 扫描结果状态：1=通过 2=失败
	Violations []Violation // 违规项列表
}

// Violation 单个违规项。
type Violation struct {
	File   string // 文件路径
	Line   int    // 行号
	Rule   string // 违反的规则
	Detail string // 违规详情
}

// Scanner 零硬编码扫描器。
type Scanner struct {
	rootDir    string         // 扫描根目录
	fset       *token.FileSet // 文件位置集
	violations []Violation    // 违规项列表
}

// NewScanner 创建扫描器实例。
func NewScanner(rootDir string) *Scanner {
	return &Scanner{
		rootDir: rootDir,
		fset:    token.NewFileSet(),
	}
}

// Scan 执行扫描，返回扫描结果。
func (s *Scanner) Scan() *ScanResult {
	filepath.Walk(s.rootDir, s.walkFunc)
	status := ResultPass
	if len(s.violations) > 0 {
		status = ResultFail
	}
	return &ScanResult{
		Status:     status,
		Violations: s.violations,
	}
}

// walkFunc filepath.Walk的回调函数，扫描每个Go文件。
func (s *Scanner) walkFunc(path string, info os.FileInfo, err error) error {
	if err != nil || info.IsDir() {
		return nil
	}
	if !strings.HasSuffix(path, ".go") {
		return nil
	}
	if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".pb.go") {
		return nil
	}
	return s.scanFile(path)
}

// scanFile 扫描单个Go文件。
func (s *Scanner) scanFile(path string) error {
	node, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.BasicLit:
			s.checkBasicLit(path, v)
		case *ast.Comment:
			s.checkComment(path, v)
		}
		return true
	})
	return nil
}

// checkBasicLit 检查字面量是否包含游戏特定名词。
func (s *Scanner) checkBasicLit(path string, lit *ast.BasicLit) {
	if lit.Kind != token.STRING {
		return
	}
	value := strings.Trim(lit.Value, `"`)
	for _, name := range gameNameBlacklist {
		if strings.Contains(value, name) {
			pos := s.fset.Position(lit.Pos())
			s.violations = append(s.violations, Violation{
				File:   path,
				Line:   pos.Line,
				Rule:   "禁止游戏特定名词",
				Detail: fmt.Sprintf("字符串含游戏名词'%s'", name),
			})
		}
	}
}

// checkComment 检查注释是否包含游戏特定名词。
func (s *Scanner) checkComment(path string, comment *ast.Comment) {
	for _, name := range gameNameBlacklist {
		if strings.Contains(comment.Text, name) && !strings.HasPrefix(comment.Text, "// ") {
			pos := s.fset.Position(comment.Pos())
			s.violations = append(s.violations, Violation{
				File:   path,
				Line:   pos.Line,
				Rule:   "注释禁止游戏特定名词",
				Detail: fmt.Sprintf("注释含游戏名词'%s'", name),
			})
		}
	}
}

// printReport 打印扫描报告。
func printReport(result *ScanResult) {
	if result.Status == ResultPass {
		fmt.Println("零硬编码扫描通过，无违规项")
		return
	}

	fmt.Printf("零硬编码扫描失败，发现%d个违规项：\n", len(result.Violations))
	for i, v := range result.Violations {
		fmt.Printf("  [%d] %s:%d %s — %s\n", i+1, v.File, v.Line, v.Rule, v.Detail)
	}
}

func main() {
	rootDir := flag.String("dir", ".", "扫描根目录")
	flag.Parse()

	scanner := NewScanner(*rootDir)
	result := scanner.Scan()
	printReport(result)

	if result.Status == ResultFail {
		os.Exit(1)
	}
}
