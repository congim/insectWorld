// 本文件定义公式AST节点类型。AST由解析器编译期生成、运行期只读，
// 求值采用直接树遍历（ADR-001 3.4执行模型，不做字节码VM）。
package formula

// exprNode AST节点接口，全部节点实现eval方法完成树遍历求值。
type exprNode interface {
	// eval 求值当前节点，返回float64；ctx携带变量表与随机源。
	eval(ctx EvalContext) (float64, error)
}

// numberNode 数字字面量节点，值为解析期转换出的浮点数。
type numberNode struct {
	value float64 // 数字字面量值，如1.5/100，配置公式允许小数（AGENTS.md规范8配置数值例外）
}

// variableNode 变量引用节点，求值期从扁平变量表按名读取。
type variableNode struct {
	name string // 变量名，如atk/def/counter/level/base，对应调用方拍平的变量表（ADR-001 3.1）
}

// unaryNode 一元运算节点，op为"-"（取负）或"not"（逻辑非）。
type unaryNode struct {
	op      string   // 一元运算符，"-"或"not"，与函数库注册名一致
	operand exprNode // 操作数表达式
}

// binaryNode 二元运算节点，op为算术/比较/逻辑运算符（"+ - * / == != > >= < <= and or"）。
type binaryNode struct {
	op    string   // 运算符文本，与函数库注册名一致，如"+"、"=="、"and"
	fn    function // 编译期从函数库解析出的运算符实现，运行期直接调用（and/or短路求值除外）
	left  exprNode // 左操作数表达式
	right exprNode // 右操作数表达式
}

// callNode 内置函数调用节点，参数表达式求值顺序从左到右。
type callNode struct {
	name string     // 函数名，如min/floor/random，与函数库注册名一致
	fn   function   // 编译期从函数库解析出的函数实现，运行期直接调用
	args []exprNode // 实参表达式列表，个数已在解析期按函数参数约束校验
}

// ifNode 条件节点，对应if(cond, t, f)语法，cond为真求值thenBranch否则求值elseBranch。
// 采用惰性求值：仅求值被选中的分支，避免未选中分支的除零等副作用（如if(x != 0, 10/x, 0)）。
type ifNode struct {
	cond       exprNode // 条件表达式，求值结果非0视为真
	thenBranch exprNode // 条件为真时求值的分支
	elseBranch exprNode // 条件为假时求值的分支
}
