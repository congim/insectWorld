#!/bin/bash
# CI 规范检查脚本：动态发现仓库中的 Go module 与 DDD 服务目录，避免新增模块逃逸检查。

set -euo pipefail

MODULES=()
EXTERNAL_MODULES=()
SERVICES=()

for module_file in server/*/go.mod; do
    [ -f "$module_file" ] || continue
    MODULES+=("${module_file%/go.mod}")
done

for module_file in tests/*/go.mod tools/go.mod; do
    [ -f "$module_file" ] || continue
    EXTERNAL_MODULES+=("${module_file%/go.mod}")
done

for service_dir in server/*; do
    [ -d "$service_dir/domain" ] || continue
    SERVICES+=("$service_dir")
done

if [ "${#MODULES[@]}" -eq 0 ]; then
    echo "FAIL: server 下未发现 Go module"
    exit 1
fi

run_in_modules() {
    local command_text="$1"
    local module_dir
    for module_dir in "${MODULES[@]}"; do
        (cd "$module_dir" && eval "$command_text")
    done
}

echo "========== CI 规范检查开始 =========="
echo "发现 ${#MODULES[@]} 个服务端 module、${#EXTERNAL_MODULES[@]} 个外部 module、${#SERVICES[@]} 个 DDD 服务目录"

echo ">>> [0/7] gofmt 检查"
for module_dir in "${MODULES[@]}" "${EXTERNAL_MODULES[@]}"; do
    gofmt_files=$(cd "$module_dir" && gofmt -l . 2>/dev/null || true)
    if [ -n "$gofmt_files" ]; then
        echo "FAIL: $module_dir 存在未格式化文件"
        echo "$gofmt_files"
        exit 1
    fi
done
echo "PASS: gofmt 检查通过"

echo ">>> [1/7] goimports 检查"
if command -v goimports >/dev/null 2>&1; then
    for module_dir in "${MODULES[@]}" "${EXTERNAL_MODULES[@]}"; do
        goimports_files=$(cd "$module_dir" && goimports -l . 2>/dev/null || true)
        if [ -n "$goimports_files" ]; then
            echo "FAIL: $module_dir 存在未整理 import 的文件"
            echo "$goimports_files"
            exit 1
        fi
    done
    echo "PASS: goimports 检查通过"
else
    echo "SKIP: goimports 未安装"
fi

echo ">>> [2/7] golangci-lint 检查"
if command -v golangci-lint >/dev/null 2>&1; then
    run_in_modules "golangci-lint run ./..."
    for module_dir in "${EXTERNAL_MODULES[@]}"; do
        (cd "$module_dir" && golangci-lint run ./...)
    done
    echo "PASS: golangci-lint 检查通过"
else
    echo "SKIP: golangci-lint 未安装"
fi

echo ">>> [3/7] 注释、字段与类型规范扫描"
for service_dir in "${SERVICES[@]}"; do
    (cd tools && go run ./spec_scanner -dir "../$service_dir")
done
(cd tools && go run ./spec_scanner -dir ../server/shared/pkg)
echo "PASS: 规范扫描通过"

echo ">>> [4/7] 表名扫描"
for service_dir in "${SERVICES[@]}"; do
    (cd tools && go run ./table_name_scanner -dir "../$service_dir")
done
echo "PASS: 表名扫描通过"

echo ">>> [5/7] DDD 依赖方向扫描"
for service_dir in "${SERVICES[@]}"; do
    (cd tools && go run ./ddd_dependency_scanner -dir "../$service_dir")
done
echo "PASS: DDD 依赖方向扫描通过"

echo ">>> [6/7] 游戏包契约验证"
(cd tools && go run ./reskin_validator -root ../gamepacks -engine-version 0.1.0)
echo "PASS: 游戏包契约验证通过"

echo ">>> [7/7] 单元测试与竞态检测"
run_in_modules "go test -race -count=1 ./..."
for module_dir in "${EXTERNAL_MODULES[@]}"; do
    (cd "$module_dir" && go test -race -count=1 ./...)
done
echo "PASS: 单元测试全部通过"

echo "========== CI 规范检查全部通过 =========="
