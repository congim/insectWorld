// Package formula 通用公式引擎，提供AST表达式解析、内置函数库与种子化确定性随机求值。
// 公式引擎将配置中的公式字符串编译为AST，运行时按输入变量与注入的随机源求值；
// 同一输入与同一随机种子必然产生完全一致的结果（战斗回放一致性依赖此特性）。
package formula

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// maxDefaultDepth 表达式最大嵌套深度默认值，超限拒绝编译，防止恶意/失控配置。
const maxDefaultDepth = 10

// EvalContext 单次求值的输入上下文，携带变量与随机源。
type EvalContext struct {
	Vars   map[string]float64 // 变量表，公式引用的变量在此读取，缺失按0处理并记录Warn
	Rand   RandSource         // 随机源，公式含random()时必须注入，缺失时random返回0并记录错误
	source string             // 原始公式字符串，错误报告与日志上下文用（引擎内部填充）
	logger *zap.Logger        // 结构化日志器，缺失变量/随机源时记录Warn/Error（引擎内部填充）
}

// Formula 编译后的公式，由Parse生成，只读且可并发求值（同一公式多实例共享）。
type Formula struct {
	source string   // 原始公式字符串，用于错误报告与日志
	root   exprNode // AST根节点，编译期生成，运行期只读
}

// FormulaEngine 公式引擎，管理公式编译缓存、内置函数库与校验参数。
type FormulaEngine struct {
	formulas map[string]*Formula // 公式ID到编译后公式的映射，key为formulas.json中的公式ID
	funcs    map[string]function // 函数名到函数实现的映射，New时注册内置函数库
	maxDepth int                 // 表达式最大嵌套深度，超限拒绝编译
	logger   *zap.Logger         // 结构化日志器，求值Warn/Error记录（默认丢弃日志，可SetLogger注入）
	mu       sync.RWMutex        // 读写锁，保护formulas映射在热更注册与按ID求值间的并发安全
}

// NewFormulaEngine 创建公式引擎实例并注册内置函数库。
func NewFormulaEngine() *FormulaEngine {
	return &FormulaEngine{
		formulas: make(map[string]*Formula),
		funcs:    registerBuiltinFuncs(),
		maxDepth: maxDefaultDepth,
		logger:   zap.NewNop(),
	}
}

// SetLogger 注入结构化日志器，用于缺失变量/随机源等求值告警记录（AGENTS.md规范7）。
// logger为nil时重置为丢弃日志，保证引擎可零配置使用。
func (e *FormulaEngine) SetLogger(logger *zap.Logger) {
	if logger == nil {
		logger = zap.NewNop()
	}
	e.logger = logger
}

// Parse 解析并编译单个公式表达式，语法/函数/深度校验任一失败返回错误。
// 配置加载阶段对全量公式调用Parse，任一失败即拒绝加载（设计书配置校验流程）。
func (e *FormulaEngine) Parse(source string) (*Formula, error) {
	return parse(source, e.funcs, e.maxDepth)
}

// Register 注册公式ID到编译后公式的映射，重复注册返回错误，避免覆盖已注册公式。
func (e *FormulaEngine) Register(formulaID string, f *Formula) error {
	if formulaID == "" {
		return fmt.Errorf("公式ID不能为空")
	}
	if f == nil {
		return fmt.Errorf("公式不能为空")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.formulas[formulaID]; exists {
		return fmt.Errorf("公式 %s 重复注册，禁止覆盖", formulaID)
	}
	e.formulas[formulaID] = f
	return nil
}

// Eval 按变量与随机源求值公式，返回float64；同一Vars+Rand必然同结果（确定性）。
// 业务离散量（兵力、资源量）由调用方用floor/round转换为int64后落库（AGENTS.md规范8）。
func (e *FormulaEngine) Eval(f *Formula, vars map[string]float64, rand RandSource) (float64, error) {
	if f == nil {
		return 0, fmt.Errorf("公式不能为空")
	}
	ctx := EvalContext{
		Vars:   vars,
		Rand:   rand,
		source: f.source,
		logger: e.logger,
	}
	return evalNode(f.root, ctx)
}

// EvalByID 按公式ID求值，公式未注册时返回错误。
func (e *FormulaEngine) EvalByID(formulaID string, vars map[string]float64, rand RandSource) (float64, error) {
	e.mu.RLock()
	f, exists := e.formulas[formulaID]
	e.mu.RUnlock()
	if !exists {
		return 0, fmt.Errorf("公式 %s 未注册", formulaID)
	}
	return e.Eval(f, vars, rand)
}
