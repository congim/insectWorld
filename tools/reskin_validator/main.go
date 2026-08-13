// Package main 游戏包契约验证工具，离线加载所有游戏包并输出可定位的结果。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"insectworld/server/shared/pkg/gamepack"
)

// 验证结果状态常量，用于进程退出码和报告展示。
const (
	ResultPass = 1 // 游戏包加载与编译通过
	ResultFail = 2 // 游戏包加载或编译失败
)

// ValidationResult 全部游戏包验证结果。
type ValidationResult struct {
	Status      int          // 总体状态：1=通过 2=失败
	PackResults []PackResult // 按游戏包目录排序的验证明细
}

// PackResult 单个游戏包验证结果。
type PackResult struct {
	PackPath string // 游戏包目录路径
	PackID   string // 编译成功后的稳定游戏包ID，失败时为空
	Status   int    // 验证状态：1=通过 2=失败
	Detail   string // 中文验证详情或具体错误链
}

// Validator 游戏包验证器，使用共享内核的真实加载与编译契约。
type Validator struct {
	root          string // 游戏包集合根目录
	engineVersion string // 当前引擎语义版本
}

// NewValidator 创建游戏包验证器。
func NewValidator(root string, engineVersion string) *Validator {
	return &Validator{root: root, engineVersion: engineVersion}
}

// Validate 验证根目录下所有包含manifest.yaml的游戏包。
func (v *Validator) Validate() *ValidationResult {
	result := &ValidationResult{Status: ResultPass}
	entries, err := os.ReadDir(v.root)
	if err != nil {
		result.Status = ResultFail
		result.PackResults = append(result.PackResults, PackResult{PackPath: v.root, Status: ResultFail, Detail: fmt.Sprintf("读取游戏包根目录失败: %v", err)})
		return result
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(v.root, entry.Name())
		if _, err := os.Stat(filepath.Join(path, "manifest.yaml")); err == nil {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	if len(paths) < 2 {
		result.Status = ResultFail
		result.PackResults = append(result.PackResults, PackResult{PackPath: v.root, Status: ResultFail, Detail: "至少需要两个游戏包验证通用契约"})
		return result
	}

	for _, path := range paths {
		pack, err := gamepack.LoadAndCompile(path, v.engineVersion)
		if err != nil {
			result.Status = ResultFail
			result.PackResults = append(result.PackResults, PackResult{PackPath: path, Status: ResultFail, Detail: err.Error()})
			continue
		}
		result.PackResults = append(result.PackResults, PackResult{
			PackPath: path,
			PackID:   pack.Manifest.ID,
			Status:   ResultPass,
			Detail:   fmt.Sprintf("编译通过：阵营=%d 资源=%d 单位=%d 建筑=%d 地图=%d", len(pack.Factions), len(pack.Resources), len(pack.Units), len(pack.Buildings), len(pack.Maps)),
		})
	}
	return result
}

func printReport(result *ValidationResult) {
	fmt.Println("=== 游戏包契约验证报告 ===")
	for _, pack := range result.PackResults {
		status := "通过"
		if pack.Status == ResultFail {
			status = "失败"
		}
		identity := pack.PackID
		if identity == "" {
			identity = pack.PackPath
		}
		fmt.Printf("%s [%s] %s\n", identity, status, pack.Detail)
	}
	if result.Status == ResultPass {
		fmt.Println("游戏包契约验证通过")
	} else {
		fmt.Println("游戏包契约验证失败")
	}
}

func main() {
	root := flag.String("root", "../gamepacks", "游戏包集合根目录")
	engineVersion := flag.String("engine-version", "0.1.0", "当前引擎语义版本")
	flag.Parse()

	result := NewValidator(*root, *engineVersion).Validate()
	printReport(result)
	if result.Status == ResultFail {
		os.Exit(1)
	}
}
