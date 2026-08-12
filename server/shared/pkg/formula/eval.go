// 本文件实现AST树遍历求值器：运行期直接遍历AST求值，不做字节码VM（ADR-001 3.4执行模型）。
// 比较/逻辑运算符返回1.0/0.0；缺失变量按0处理并记录Warn；未注入随机源时random返回0并记录Error。
package formula

import (
	"fmt"

	"go.uber.org/zap"
)

// 比较/逻辑运算返回值的真/假常量（ADR-001 3.2：比较/逻辑运算符返回1.0/0.0）。
const (
	floatTrue  = 1.0 // 比较/逻辑为真时返回的值
	floatFalse = 0.0 // 比较/逻辑为假时返回的值
)

// evalNode 求值AST根节点，是引擎Eval的底层入口；错误统一包裹公式源码上下文。
func evalNode(node exprNode, ctx EvalContext) (float64, error) {
	v, err := node.eval(ctx)
	if err != nil {
		return 0, fmt.Errorf("公式 %q 求值失败: %w", ctx.source, err)
	}
	return v, nil
}

// boolToFloat 布尔值转浮点：true返回1.0，false返回0.0（比较/逻辑运算符返回值语义）。
func boolToFloat(b bool) float64 {
	if b {
		return floatTrue
	}
	return floatFalse
}

// warnf 记录Warn级别结构化日志，logger未注入时静默丢弃（测试/无日志场景）。
func (c EvalContext) warnf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Warn(fmt.Sprintf(format, args...), zap.String("formula_source", c.source))
	}
}

// errorf 记录Error级别结构化日志，logger未注入时静默丢弃（测试/无日志场景）。
func (c EvalContext) errorf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Error(fmt.Sprintf(format, args...), zap.String("formula_source", c.source))
	}
}

// randFloat64 从注入的随机源取值，未注入时记录Error并返回0（ADR-001 3.6：缺失时random返回0并记录错误）。
func (c EvalContext) randFloat64() float64 {
	if c.Rand == nil {
		c.errorf("公式使用random但未注入随机源，random()返回0，请注入RandSource")
		return 0
	}
	return c.Rand.Float64()
}

// eval 数字字面量节点求值，直接返回编译期解析出的值。
func (n *numberNode) eval(_ EvalContext) (float64, error) {
	return n.value, nil
}

// eval 变量引用节点求值，从变量表按名读取；缺失按0处理并记录Warn（ADR-001 3.6语义）。
func (n *variableNode) eval(ctx EvalContext) (float64, error) {
	v, ok := ctx.Vars[n.name]
	if !ok {
		ctx.warnf("变量 %s 缺失，按0处理，请检查变量表是否拍平该前缀变量", n.name)
		return 0, nil
	}
	return v, nil
}

// eval 一元运算节点求值："-"取负，not按非0为真取逻辑非（返回1.0/0.0）。
func (n *unaryNode) eval(ctx EvalContext) (float64, error) {
	v, err := n.operand.eval(ctx)
	if err != nil {
		return 0, err
	}
	switch n.op {
	case opSub:
		return -v, nil
	case opNot:
		return boolToFloat(v == 0), nil
	default:
		return 0, fmt.Errorf("未知一元运算符 %s", n.op)
	}
}

// eval 二元运算节点求值。
// and/or采用短路求值：左侧已能决定结果时不求值右侧，避免未达分支的除零等副作用；
// 其余算术/比较运算符两侧均需求值，直接调用函数库实现（与函数库注册名一一对应）。
func (n *binaryNode) eval(ctx EvalContext) (float64, error) {
	// 短路求值分支（and：左假则结果假；or：左真则结果真）
	if n.op == opAnd || n.op == opOr {
		left, err := n.left.eval(ctx)
		if err != nil {
			return 0, err
		}
		if n.op == opAnd && left == 0 {
			return floatFalse, nil
		}
		if n.op == opOr && left != 0 {
			return floatTrue, nil
		}
		right, err := n.right.eval(ctx)
		if err != nil {
			return 0, err
		}
		return boolToFloat(right != 0), nil
	}
	// 普通二元运算（算术/比较）
	left, err := n.left.eval(ctx)
	if err != nil {
		return 0, err
	}
	right, err := n.right.eval(ctx)
	if err != nil {
		return 0, err
	}
	return n.fn.fn([]float64{left, right}, ctx)
}

// eval 函数调用节点求值，从左到右求值全部实参后调用函数实现。
func (n *callNode) eval(ctx EvalContext) (float64, error) {
	args := make([]float64, 0, len(n.args))
	for _, arg := range n.args {
		v, err := arg.eval(ctx)
		if err != nil {
			return 0, err
		}
		args = append(args, v)
	}
	return n.fn.fn(args, ctx)
}

// eval 条件节点求值，先求值条件，再惰性求值被选中的分支（ADR-001 3.2 if语义）。
func (n *ifNode) eval(ctx EvalContext) (float64, error) {
	cond, err := n.cond.eval(ctx)
	if err != nil {
		return 0, err
	}
	if cond != 0 {
		return n.thenBranch.eval(ctx)
	}
	return n.elseBranch.eval(ctx)
}
