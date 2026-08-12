// Package main DDD依赖方向校验工具，落地AGENTS.md规范3（DDD架构）。
// 校验依赖方向：interfaces→application→domain←infrastructure。
// domain层零外部依赖，application层不直接import infrastructure。
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// DependencyViolation 依赖方向违规记录。
type DependencyViolation struct {
	File       string // 文件路径
	ImportPath string // 违规import路径
	Layer      string // 当前文件所在层
	Message    string // 违规描述
}

// DDDScanner DDD依赖方向扫描器。
type DDDScanner struct {
	violations []DependencyViolation
	fset       *token.FileSet
}

// NewDDDScanner 创建DDD依赖方向扫描器实例。
func NewDDDScanner() *DDDScanner {
	return &DDDScanner{
		fset: token.NewFileSet(),
	}
}

// Scan 扫描指定目录下的所有Go源文件。
func (s *DDDScanner) Scan(dir string) error {
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

// scanFile 扫描单个Go源文件的import声明。
func (s *DDDScanner) scanFile(path string) error {
	node, err := parser.ParseFile(s.fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}

	layer := detectLayer(path)
	if layer == "" {
		return nil
	}

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(importPath, "insectworld/server/") {
			continue
		}
		if strings.HasPrefix(importPath, "insectworld/server/shared/") ||
			strings.HasPrefix(importPath, "insectworld/tests/integration") {
			continue
		}

		importLayer := detectLayerFromImport(importPath)
		if importLayer == "" {
			continue
		}

		if s.isViolation(layer, importLayer, importPath) {
			s.violations = append(s.violations, DependencyViolation{
				File:       path,
				ImportPath: importPath,
				Layer:      layer,
				Message:    fmt.Sprintf("%s层不得依赖%s层", layer, importLayer),
			})
		}
	}
	return nil
}

// detectLayer 从文件路径检测DDD层级。
func detectLayer(path string) string {
	path = filepath.ToSlash(path)
	if strings.Contains(path, "/domain/") {
		return "domain"
	}
	if strings.Contains(path, "/application/") {
		return "application"
	}
	if strings.Contains(path, "/infrastructure/") {
		return "infrastructure"
	}
	if strings.Contains(path, "/interfaces/") {
		return "interfaces"
	}
	return ""
}

// detectLayerFromImport 从import路径检测DDD层级。
func detectLayerFromImport(importPath string) string {
	if strings.Contains(importPath, "/domain/") {
		return "domain"
	}
	if strings.Contains(importPath, "/application/") {
		return "application"
	}
	if strings.Contains(importPath, "/infrastructure/") {
		return "infrastructure"
	}
	if strings.Contains(importPath, "/interfaces/") {
		return "interfaces"
	}
	return ""
}

// isViolation 判断依赖方向是否违规。
// 合规方向：interfaces→application→domain←infrastructure
func (s *DDDScanner) isViolation(currentLayer, importLayer, importPath string) bool {
	switch currentLayer {
	case "domain":
		// domain层零外部依赖：不得依赖application/infrastructure/interfaces
		return importLayer != "domain"
	case "application":
		// application层不得直接依赖infrastructure
		return importLayer == "infrastructure"
	case "interfaces":
		// interfaces层不得直接依赖infrastructure（应通过application/domain）。
		// interfaces→domain允许：interfaces需要domain的错误码/类型做协议转换。
		return importLayer == "infrastructure"
	default:
		return false
	}
}

// Report 输出违规报告。
func (s *DDDScanner) Report() int {
	if len(s.violations) == 0 {
		fmt.Println("DDD依赖方向校验通过")
		return 0
	}
	fmt.Printf("发现%d个DDD依赖方向违规：\n", len(s.violations))
	for _, v := range s.violations {
		fmt.Printf("  %s [%s] import %s: %s\n", v.File, v.Layer, v.ImportPath, v.Message)
	}
	return 1
}

func main() {
	dir := flag.String("dir", ".", "扫描目录")
	flag.Parse()

	scanner := NewDDDScanner()
	if err := scanner.Scan(*dir); err != nil {
		fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
		os.Exit(2)
	}
	os.Exit(scanner.Report())
}
