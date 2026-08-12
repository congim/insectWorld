// 本文件实现递归下降语法解析器：按ADR-001 3.1的EBNF语法解析token流生成AST，
// 同时完成编译期校验（函数是否存在/参数个数/嵌套深度，对应ADR-001 3.4的Validate步骤）。
// 运算符优先级（高→低）：() > 一元- / not > * / > + - > 比较 > and > or。
package formula

import "fmt"

// parse 解析并编译单个公式表达式，返回AST根节点；
// 语法/函数/参数个数/嵌套深度任一校验失败返回带位置错误，配置加载阶段据此拒绝加载。
// funcs为函数库（函数名到实现的映射），maxDepth为表达式最大嵌套深度。
func parse(source string, funcs map[string]function, maxDepth int) (*Formula, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	p := &parser{
		source:   source,
		tokens:   tokens,
		funcs:    funcs,
		maxDepth: maxDepth,
	}
	root, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	// 表达式解析结束后必须到达输入末尾，剩余token视为语法错误（如"1 2"）
	if tok := p.peek(); tok.typ != tokenEOF {
		return nil, p.errorAt(tok, fmt.Sprintf("多余的token %q", tok.text))
	}
	return &Formula{source: source, root: root}, nil
}

// parser 递归下降解析器，维护token流游标与编译期校验上下文。
type parser struct {
	source   string              // 原始公式字符串，错误报告用
	tokens   []token             // 词法分析输出的token流，以tokenEOF结尾
	pos      int                 // 当前token索引
	funcs    map[string]function // 函数库，校验函数名/运算符存在性与参数个数
	maxDepth int                 // 表达式最大嵌套深度，超限拒绝编译（防恶意/失控配置）
}

// peek 返回当前token（不消费），游标越界时返回tokenEOF。
func (p *parser) peek() token {
	return p.tokens[p.pos]
}

// next 消费并返回当前token，游标后移一位。
func (p *parser) next() token {
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

// errorAt 生成定位到字符位置的语法错误，格式：公式+第N字符+原因，供配置加载输出文件+公式ID+原因+修复建议。
func (p *parser) errorAt(tok token, reason string) error {
	return fmt.Errorf("公式 %q 第%d字符位置: %s", p.source, tok.pos+1, reason)
}

// parseExpression 解析完整表达式，depth为当前嵌套深度（括号/函数调用/一元运算符递归时递增）。
func (p *parser) parseExpression(depth int) (exprNode, error) {
	if depth > p.maxDepth {
		return nil, fmt.Errorf("公式 %q: 表达式嵌套深度超过限制（最大%d层，请拆分公式）", p.source, p.maxDepth)
	}
	return p.parseOr(depth)
}

// parseOr 解析逻辑或表达式：and_expr ( "or" and_expr )*，优先级最低。
func (p *parser) parseOr(depth int) (exprNode, error) {
	left, err := p.parseAnd(depth)
	if err != nil {
		return nil, err
	}
	for p.peek().typ == tokenOr {
		p.next()
		right, err := p.parseAnd(depth)
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: opOr, fn: p.funcs[opOr], left: left, right: right}
	}
	return left, nil
}

// parseAnd 解析逻辑与表达式：cmp_expr ( "and" cmp_expr )*，优先级高于or。
func (p *parser) parseAnd(depth int) (exprNode, error) {
	left, err := p.parseCmp(depth)
	if err != nil {
		return nil, err
	}
	for p.peek().typ == tokenAnd {
		p.next()
		right, err := p.parseCmp(depth)
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: opAnd, fn: p.funcs[opAnd], left: left, right: right}
	}
	return left, nil
}

// parseCmp 解析比较表达式：add_expr ( ("==" | "!=" | ">" | ">=" | "<" | "<=") add_expr )?
// 按EBNF比较运算是可选的单个（非结合），连续比较如"a < b < c"视为语法错误。
func (p *parser) parseCmp(depth int) (exprNode, error) {
	left, err := p.parseAdd(depth)
	if err != nil {
		return nil, err
	}
	op, ok := cmpTokenToOp[p.peek().typ]
	if !ok {
		return left, nil
	}
	p.next()
	right, err := p.parseAdd(depth)
	if err != nil {
		return nil, err
	}
	return &binaryNode{op: op, fn: p.funcs[op], left: left, right: right}, nil
}

// cmpTokenToOp 比较运算符token类型到运算符名的映射，与函数库注册名一致。
var cmpTokenToOp = map[tokenType]string{
	tokenEq: opEq, tokenNe: opNe,
	tokenGt: opGt, tokenGe: opGe,
	tokenLt: opLt, tokenLe: opLe,
}

// parseAdd 解析加减表达式：mul_expr ( ("+" | "-") mul_expr )*，左结合。
func (p *parser) parseAdd(depth int) (exprNode, error) {
	left, err := p.parseMul(depth)
	if err != nil {
		return nil, err
	}
	for {
		op := ""
		switch p.peek().typ {
		case tokenPlus:
			op = opAdd
		case tokenMinus:
			op = opSub
		default:
			return left, nil
		}
		p.next()
		right, err := p.parseMul(depth)
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: op, fn: p.funcs[op], left: left, right: right}
	}
}

// parseMul 解析乘除表达式：unary_expr ( ("*" | "/") unary_expr )*，左结合，优先级高于加减。
func (p *parser) parseMul(depth int) (exprNode, error) {
	left, err := p.parseUnary(depth)
	if err != nil {
		return nil, err
	}
	for {
		op := ""
		switch p.peek().typ {
		case tokenStar:
			op = opMul
		case tokenSlash:
			op = opDiv
		default:
			return left, nil
		}
		p.next()
		right, err := p.parseUnary(depth)
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: op, fn: p.funcs[op], left: left, right: right}
	}
}

// parseUnary 解析一元表达式：("-" | "not") unary_expr | primary，一元递归计入嵌套深度。
func (p *parser) parseUnary(depth int) (exprNode, error) {
	if depth > p.maxDepth {
		return nil, fmt.Errorf("公式 %q: 表达式嵌套深度超过限制（最大%d层，请拆分公式）", p.source, p.maxDepth)
	}
	switch p.peek().typ {
	case tokenMinus:
		p.next()
		operand, err := p.parseUnary(depth + 1)
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: opSub, operand: operand}, nil
	case tokenNot:
		p.next()
		operand, err := p.parseUnary(depth + 1)
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: opNot, operand: operand}, nil
	default:
		return p.parsePrimary(depth)
	}
}

// parsePrimary 解析基本表达式：number | variable | func_call | "(" expression ")"。
func (p *parser) parsePrimary(depth int) (exprNode, error) {
	tok := p.peek()
	switch tok.typ {
	case tokenNumber:
		p.next()
		return &numberNode{value: tok.value}, nil
	case tokenIdent:
		p.next()
		// 标识符后紧跟左括号视为函数调用，否则视为变量引用
		if p.peek().typ == tokenLParen {
			return p.parseCall(tok, depth)
		}
		return &variableNode{name: tok.text}, nil
	case tokenLParen:
		p.next()
		node, err := p.parseExpression(depth + 1)
		if err != nil {
			return nil, err
		}
		if p.peek().typ != tokenRParen {
			return nil, p.errorAt(p.peek(), "缺少右括号 )")
		}
		p.next()
		return node, nil
	default:
		if tok.typ == tokenEOF {
			return nil, p.errorAt(tok, "表达式不完整，期望操作数")
		}
		return nil, p.errorAt(tok, fmt.Sprintf("意外的token %q", tok.text))
	}
}

// parseCall 解析函数调用：identifier "(" [expression ("," expression)*] ")"，
// 完成函数存在性/参数个数校验；if函数为惰性条件函数，编译为条件节点（ifNode）。
func (p *parser) parseCall(nameTok token, depth int) (exprNode, error) {
	fn, ok := p.funcs[nameTok.text]
	if !ok {
		return nil, p.errorAt(nameTok, fmt.Sprintf("函数 %s 未定义，请检查函数名拼写（内置函数：min/max/floor/ceil/round/random/clamp/abs/if）", nameTok.text))
	}
	// 消费左括号
	p.next()
	args := make([]exprNode, 0, 2)
	for p.peek().typ != tokenRParen {
		arg, err := p.parseExpression(depth + 1)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek().typ != tokenComma {
			break
		}
		p.next()
		// 拒绝尾随逗号（如"random(1,)"），参数列表每项必须是表达式
		if p.peek().typ == tokenRParen {
			return nil, p.errorAt(p.peek(), fmt.Sprintf("函数 %s 参数列表末尾存在多余逗号", nameTok.text))
		}
	}
	if p.peek().typ != tokenRParen {
		return nil, p.errorAt(p.peek(), fmt.Sprintf("函数 %s 缺少右括号 )", nameTok.text))
	}
	p.next() // 消费右括号
	// 参数个数校验（ADR-001 3.4 Validate：参数个数）
	if len(args) < fn.minArgs || len(args) > fn.maxArgs {
		return nil, p.errorAt(nameTok, fmt.Sprintf("函数 %s 参数个数错误：期望%s，实际%d个", nameTok.text, arityDesc(fn), len(args)))
	}
	// if为惰性条件函数（lazy标记），编译为条件节点实现惰性求值
	if fn.lazy {
		return &ifNode{cond: args[0], thenBranch: args[1], elseBranch: args[2]}, nil
	}
	return &callNode{name: nameTok.text, fn: fn, args: args}, nil
}
