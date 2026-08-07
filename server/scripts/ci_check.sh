#!/bin/bash
# CI规范检查脚本，落地AGENTS.md全部9条规范的CI静态检查。
# 适配微服务多module结构，遍历9个module目录逐一检查。
# 在CI流水线中执行，任一检查失败即阻断合入。

set -e

# 全部Go module目录（shared + 8个服务 + 集成测试）
MODULES=("shared" "world" "combat" "economy" "social" "operation" "gateway" "config" "persist" "integration")

# 全部服务目录（用于规范扫描）
SERVICES=("world" "combat" "economy" "social" "operation" "gateway" "config" "persist")

# run_check_per_module 对每个Go module目录执行指定命令。
# 参数1: 检查名称（用于日志输出）
# 参数2: 要执行的命令（在module目录下执行）
run_check_per_module() {
    local check_name="$1"
    local cmd="$2"
    for mod in "${MODULES[@]}"; do
        (cd "$mod" && eval "$cmd")
    done
}

# run_check_per_service 对每个服务目录执行指定命令。
# 参数1: 检查名称（用于日志输出）
# 参数2: 要执行的命令（在服务目录下执行，通过$svc变量引用服务名）
run_check_per_service() {
    local check_name="$1"
    local cmd="$2"
    for svc in "${SERVICES[@]}"; do
        eval "$cmd"
    done
}

echo "========== CI规范检查开始 =========="

# 0. 代码格式化检查（规范9）
echo ">>> [0/6] gofmt检查..."
for mod in "${MODULES[@]}"; do
    gofmt_files=$(cd "$mod" && gofmt -l . 2>/dev/null || true)
    if [ -n "$gofmt_files" ]; then
        echo "FAIL: $mod gofmt检查未通过，以下文件需格式化:"
        echo "$gofmt_files"
        exit 1
    fi
done
echo "PASS: gofmt检查通过"

# 1. goimports检查（规范9）
echo ">>> [1/6] goimports检查..."
if command -v goimports &> /dev/null; then
    for mod in "${MODULES[@]}"; do
        goimports_files=$(cd "$mod" && goimports -l . 2>/dev/null || true)
        if [ -n "$goimports_files" ]; then
            echo "FAIL: $mod goimports检查未通过"
            echo "$goimports_files"
            exit 1
        fi
    done
    echo "PASS: goimports检查通过"
else
    echo "SKIP: goimports未安装，跳过"
fi

# 2. golangci-lint检查（规范3/5/7/9）
echo ">>> [2/6] golangci-lint检查..."
if command -v golangci-lint &> /dev/null; then
    run_check_per_module "golangci-lint" "golangci-lint run ./..."
    echo "PASS: golangci-lint检查通过"
else
    echo "SKIP: golangci-lint未安装，跳过"
fi

# 3. 自定义规范扫描（规范1/5/6/8/9）
echo ">>> [3/6] 规范扫描（spec_scanner）..."
run_check_per_service "spec_scanner" "(cd tools && go run ./spec_scanner -dir \"../\$svc\")"
(cd tools && go run ./spec_scanner -dir ../shared/pkg)
echo "PASS: 规范扫描通过"

# 4. 表名t_前缀扫描（规范2）
echo ">>> [4/6] 表名t_前缀扫描..."
run_check_per_service "table_name_scanner" "(cd tools && go run ./table_name_scanner -dir \"../\$svc\")"
echo "PASS: 表名扫描通过"

# 5. DDD依赖方向校验（规范3）
echo ">>> [5/6] DDD依赖方向校验..."
run_check_per_service "ddd_dependency_scanner" "(cd tools && go run ./ddd_dependency_scanner -dir \"../\$svc\")"
echo "PASS: DDD依赖方向校验通过"

# 6. 单元测试+竞态检测（规范9）
echo ">>> [6/6] 单元测试（go test -race）..."
run_check_per_module "unit_test" "go test -race -count=1 ./..."
echo "PASS: 单元测试全部通过"

echo "========== CI规范检查全部通过 =========="
