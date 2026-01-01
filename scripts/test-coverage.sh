#!/bin/bash
# 测试覆盖率分析脚本
# 用途：运行测试、生成覆盖率报告、生成 HTML 可视化

set -e

PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$PROJECT_ROOT"

echo "==> 运行测试并生成覆盖率报告..."
go test -mod=mod -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... 2>&1 | tee test-output.log

echo ""
echo "==> 生成 HTML 覆盖率报告..."
go tool cover -html=coverage.out -o coverage.html

echo ""
echo "==> 覆盖率统计："
echo ""

# 提取覆盖率数据并生成摘要
cat coverage.out | grep -v "mode:" | awk '
BEGIN {
    total_stmts = 0
    covered_stmts = 0
}
{
    # 解析每一行: file.go:start.col,end.col num_stmts count
    split($2, range, ",")
    stmts = $3
    count = $4

    total_stmts += stmts
    if (count > 0) {
        covered_stmts += stmts
    }
}
END {
    if (total_stmts > 0) {
        coverage = (covered_stmts / total_stmts) * 100
        printf "总语句数: %d\n", total_stmts
        printf "已覆盖语句数: %d\n", covered_stmts
        printf "总体覆盖率: %.2f%%\n", coverage
    }
}
'

echo ""
echo "==> 按包统计覆盖率（Top 20）："
echo ""

# 按包统计覆盖率
cat coverage.out | grep -v "mode:" | awk -F':' '
{
    pkg = $1
    # 提取覆盖率信息
    split($0, parts, " ")
    stmts = parts[3]
    count = parts[4]

    pkg_total[pkg] += stmts
    if (count > 0) {
        pkg_covered[pkg] += stmts
    }
}
END {
    for (pkg in pkg_total) {
        if (pkg_total[pkg] > 0) {
            coverage = (pkg_covered[pkg] / pkg_total[pkg]) * 100
            printf "%s\t%.2f%%\n", pkg, coverage
        }
    }
}
' | sort -t$'\t' -k2 -rn | head -20

echo ""
echo "==> 低覆盖率文件（<50%）："
echo ""

cat coverage.out | grep -v "mode:" | awk -F':' '
{
    file = $1
    split($0, parts, " ")
    stmts = parts[3]
    count = parts[4]

    file_total[file] += stmts
    if (count > 0) {
        file_covered[file] += stmts
    }
}
END {
    for (file in file_total) {
        if (file_total[file] > 0) {
            coverage = (file_covered[file] / file_total[file]) * 100
            if (coverage < 50) {
                printf "%s\t%.2f%%\n", file, coverage
            }
        }
    }
}
' | sort -t$'\t' -k2 -n | head -20

echo ""
echo "==> 报告已生成："
echo "  - coverage.out    (覆盖率数据)"
echo "  - coverage.html   (HTML 可视化报告)"
echo "  - test-output.log (测试输出日志)"
echo ""
echo "打开 HTML 报告: open coverage.html"
