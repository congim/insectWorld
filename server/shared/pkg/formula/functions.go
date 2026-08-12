// 本文件实现内置函数库：命名函数（min/max/floor/ceil/round/random/clamp/abs/if）
// 与比较/逻辑运算符统一登记在函数表中（ADR-001 3.2），供解析期校验函数名与参数个数、
// 运行期求值调用；运算符名称常量就近归属本文件（AGENTS.md规范1）。
package formula

import (
	"fmt"
	"math"
)

// 运算符名称常量，与函数库注册名保持一致（就近归属：运算符统一在函数库注册）。
const (
	opAdd = "+"   // 加法运算符
	opSub = "-"   // 减法/取负运算符
	opMul = "*"   // 乘法运算符
	opDiv = "/"   // 除法运算符
	opEq  = "=="  // 相等比较运算符
	opNe  = "!="  // 不等比较运算符
	opGt  = ">"   // 大于比较运算符
	opGe  = ">="  // 大于等于比较运算符
	opLt  = "<"   // 小于比较运算符
	opLe  = "<="  // 小于等于比较运算符
	opAnd = "and" // 逻辑与运算符
	opOr  = "or"  // 逻辑或运算符
	opNot = "not" // 逻辑非运算符
)

// function 函数库条目，描述函数名、参数个数约束与求值逻辑。
type function struct {
	name    string                                                 // 函数名，与公式中的调用名一致（运算符使用符号名如"=="）
	minArgs int                                                    // 最少参数个数，解析期校验实参数量用
	maxArgs int                                                    // 最多参数个数，解析期校验实参数量用
	lazy    bool                                                   // 是否惰性求值，仅if函数为true（编译期转为条件节点ifNode）
	fn      func(args []float64, ctx EvalContext) (float64, error) // 求值逻辑，args为已求值的实参
}

// registerBuiltinFuncs 注册全部内置函数与比较/逻辑运算符，返回函数名到条目的映射。
// 函数名与运算符名统一登记，作为解析期"函数是否存在/参数个数"校验的唯一来源（ADR-001 3.4）。
func registerBuiltinFuncs() map[string]function {
	funcs := make(map[string]function)
	reg := func(f function) { funcs[f.name] = f }
	// 命名函数（ADR-001 3.2函数库清单）
	reg(function{name: "min", minArgs: 2, maxArgs: 2, fn: builtinMin})                      // 取较小值，如赛季遗产保留量min(原值×keep_ratio, keep_amount)
	reg(function{name: "max", minArgs: 2, maxArgs: 2, fn: builtinMax})                      // 取较大值，如伤害下限max(atk×counter-def, 1)
	reg(function{name: "floor", minArgs: 1, maxArgs: 1, fn: builtinFloor})                  // 向下取整，离散业务量结算
	reg(function{name: "ceil", minArgs: 1, maxArgs: 1, fn: builtinCeil})                    // 向上取整，需求/消耗取整
	reg(function{name: "round", minArgs: 1, maxArgs: 1, fn: builtinRound})                  // 四舍五入取整，展示/结算精度处理
	reg(function{name: "random", minArgs: 0, maxArgs: 2, fn: builtinRandom})                // 均匀随机：random()→[0,1) random(n)→[0,n) random(a,b)→[a,b)，从注入的RandSource取值
	reg(function{name: "clamp", minArgs: 3, maxArgs: 3, fn: builtinClamp})                  // 钳制到[lo,hi]，属性钳制/资源上限
	reg(function{name: "abs", minArgs: 1, maxArgs: 1, fn: builtinAbs})                      // 绝对值，差值/距离计算
	reg(function{name: "if", minArgs: 3, maxArgs: 3, lazy: true, fn: builtinIfUnreachable}) // 条件选择if(cond,t,f)，惰性求值只算被选中分支
	// 算术运算符（ADR-001 3.1四则运算），与比较/逻辑运算符统一登记在函数表
	reg(function{name: opAdd, minArgs: 2, maxArgs: 2, fn: builtinArith(opAdd)}) // 加法
	reg(function{name: opSub, minArgs: 2, maxArgs: 2, fn: builtinArith(opSub)}) // 减法
	reg(function{name: opMul, minArgs: 2, maxArgs: 2, fn: builtinArith(opMul)}) // 乘法
	reg(function{name: opDiv, minArgs: 2, maxArgs: 2, fn: builtinArith(opDiv)}) // 除法，除数为0返回错误
	// 比较运算符，返回1.0/0.0，配合if使用
	reg(function{name: opEq, minArgs: 2, maxArgs: 2, fn: builtinCompare(opEq)}) // 相等比较，如种族==1
	reg(function{name: opNe, minArgs: 2, maxArgs: 2, fn: builtinCompare(opNe)}) // 不等比较
	reg(function{name: opGt, minArgs: 2, maxArgs: 2, fn: builtinCompare(opGt)}) // 大于比较
	reg(function{name: opGe, minArgs: 2, maxArgs: 2, fn: builtinCompare(opGe)}) // 大于等于比较
	reg(function{name: opLt, minArgs: 2, maxArgs: 2, fn: builtinCompare(opLt)}) // 小于比较
	reg(function{name: opLe, minArgs: 2, maxArgs: 2, fn: builtinCompare(opLe)}) // 小于等于比较
	// 逻辑运算符，返回1.0/0.0，配合if使用（语法上and/or/not为保留关键字，不可作为变量名）
	reg(function{name: opAnd, minArgs: 2, maxArgs: 2, fn: builtinLogical(opAnd)}) // 逻辑与，短路求值见binaryNode.eval
	reg(function{name: opOr, minArgs: 2, maxArgs: 2, fn: builtinLogical(opOr)})   // 逻辑或，短路求值见binaryNode.eval
	reg(function{name: opNot, minArgs: 1, maxArgs: 1, fn: builtinLogical(opNot)}) // 逻辑非
	return funcs
}

// arityDesc 生成参数个数约束的可读描述，如"2个"或"0~2个"，用于解析期错误报告。
func arityDesc(fn function) string {
	if fn.minArgs == fn.maxArgs {
		return fmt.Sprintf("%d个", fn.minArgs)
	}
	return fmt.Sprintf("%d~%d个", fn.minArgs, fn.maxArgs)
}

// builtinMin 返回两参数中的较小值。
func builtinMin(args []float64, _ EvalContext) (float64, error) {
	if args[0] < args[1] {
		return args[0], nil
	}
	return args[1], nil
}

// builtinMax 返回两参数中的较大值。
func builtinMax(args []float64, _ EvalContext) (float64, error) {
	if args[0] > args[1] {
		return args[0], nil
	}
	return args[1], nil
}

// builtinFloor 向下取整，返回不大于参数的最大整数。
func builtinFloor(args []float64, _ EvalContext) (float64, error) {
	return math.Floor(args[0]), nil
}

// builtinCeil 向上取整，返回不小于参数的最小整数。
func builtinCeil(args []float64, _ EvalContext) (float64, error) {
	return math.Ceil(args[0]), nil
}

// builtinRound 四舍五入取整，返回最接近参数的整数（.5按远离零方向舍入，与math.Round一致）。
func builtinRound(args []float64, _ EvalContext) (float64, error) {
	return math.Round(args[0]), nil
}

// builtinRandom 均匀随机数生成：random()→[0,1)；random(n)→[0,n)；random(a,b)→[a,b)。
// 从注入的RandSource取值，未注入随机源时返回0并记录Error（ADR-001 3.6语义）。
func builtinRandom(args []float64, ctx EvalContext) (float64, error) {
	switch len(args) {
	case 0:
		return ctx.randFloat64(), nil
	case 1:
		return ctx.randFloat64() * args[0], nil
	case 2:
		// 区间随机：a + r*(b-a)，r∈[0,1)，结果∈[a,b)
		return args[0] + ctx.randFloat64()*(args[1]-args[0]), nil
	default:
		return 0, fmt.Errorf("random参数个数错误：期望0~2个，实际%d个", len(args))
	}
}

// builtinClamp 将参数钳制到[lo,hi]区间，超出下界返回lo，超出上界返回hi。
func builtinClamp(args []float64, _ EvalContext) (float64, error) {
	x, lo, hi := args[0], args[1], args[2]
	if x < lo {
		return lo, nil
	}
	if x > hi {
		return hi, nil
	}
	return x, nil
}

// builtinAbs 返回参数的绝对值。
func builtinAbs(args []float64, _ EvalContext) (float64, error) {
	return math.Abs(args[0]), nil
}

// builtinIfUnreachable if函数的占位实现。if在解析期转为惰性条件节点（ifNode），
// 不会通过函数表调用本实现；保留占位保持函数库注册完整（ADR-001 3.2 if条目）。
func builtinIfUnreachable(_ []float64, _ EvalContext) (float64, error) {
	return 0, fmt.Errorf("if为惰性条件函数，应编译为条件节点求值，不应直接调用")
}

// builtinArith 算术运算实现工厂，按op计算左右操作数；除法除数为0时返回错误（配置防御，
// 避免Inf漂移到下游int64离散量转换，AGENTS.md规范8）。
func builtinArith(op string) func(args []float64, ctx EvalContext) (float64, error) {
	return func(args []float64, _ EvalContext) (float64, error) {
		l, r := args[0], args[1]
		switch op {
		case opAdd:
			return l + r, nil
		case opSub:
			return l - r, nil
		case opMul:
			return l * r, nil
		case opDiv:
			if r == 0 {
				return 0, fmt.Errorf("除数为0（运算 %v / %v），请检查公式分母表达式", l, r)
			}
			return l / r, nil
		default:
			return 0, fmt.Errorf("未知算术运算符 %s", op)
		}
	}
}

// builtinCompare 比较运算实现工厂，按op比较左右操作数返回1.0/0.0（ADR-001 3.2比较运算符语义）。
func builtinCompare(op string) func(args []float64, ctx EvalContext) (float64, error) {
	return func(args []float64, _ EvalContext) (float64, error) {
		l, r := args[0], args[1]
		switch op {
		case opEq:
			return boolToFloat(l == r), nil
		case opNe:
			return boolToFloat(l != r), nil
		case opGt:
			return boolToFloat(l > r), nil
		case opGe:
			return boolToFloat(l >= r), nil
		case opLt:
			return boolToFloat(l < r), nil
		case opLe:
			return boolToFloat(l <= r), nil
		default:
			return 0, fmt.Errorf("未知比较运算符 %s", op)
		}
	}
}

// builtinLogical 逻辑运算实现工厂，按op对参数做真值判断（非0为真）返回1.0/0.0。
// 注意：and/or在AST求值时走短路路径（binaryNode.eval），本实现为函数库注册完整性保留。
func builtinLogical(op string) func(args []float64, ctx EvalContext) (float64, error) {
	return func(args []float64, _ EvalContext) (float64, error) {
		switch op {
		case opAnd:
			return boolToFloat(args[0] != 0 && args[1] != 0), nil
		case opOr:
			return boolToFloat(args[0] != 0 || args[1] != 0), nil
		case opNot:
			return boolToFloat(args[0] == 0), nil
		default:
			return 0, fmt.Errorf("未知逻辑运算符 %s", op)
		}
	}
}
