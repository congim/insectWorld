// 本文件实现公式表达式的词法分析：将源字符串切分为token流。
// 每个token携带字节偏移，供语法错误定位到字符位置（ADR-001 E1-S1验收要求）。
package formula

import (
	"fmt"
	"strconv"
)

// tokenType token类型枚举，覆盖公式语法（ADR-001 3.1 EBNF）的全部词法单元。
type tokenType int

const (
	tokenEOF     tokenType = iota // 输入结束
	tokenNumber                   // 数字字面量（整数或小数）
	tokenIdent                    // 标识符（变量名或函数名，字母开头含字母数字下划线）
	tokenAnd                      // 逻辑与关键字 and
	tokenOr                       // 逻辑或关键字 or
	tokenNot                      // 逻辑非关键字 not
	tokenPlus                     // 加号 +
	tokenMinus                    // 减号/负号 -
	tokenStar                     // 乘号 *
	tokenSlash                    // 除号 /
	tokenLParen                   // 左括号 (
	tokenRParen                   // 右括号 )
	tokenComma                    // 逗号 ,
	tokenEq                       // 相等比较 ==
	tokenNe                       // 不等比较 !=
	tokenGt                       // 大于 >
	tokenGe                       // 大于等于 >=
	tokenLt                       // 小于 <
	tokenLe                       // 小于等于 <=
	tokenIllegal                  // 非法字符（预留，实际在词法阶段直接报错）
)

// token 词法单元，携带类型、原始文本、字节偏移与数字字面量值。
type token struct {
	typ   tokenType // token类型，见tokenType枚举
	text  string    // 原始文本（标识符名/数字文本/运算符符号），错误报告用
	pos   int       // 在源串中的字节偏移（从0起），错误定位到字符位置用
	value float64   // 数字字面量的解析值，非数字token恒为0
}

// tokenTypeName token类型的可读名称，用于错误报告展示实际遇到的词法单元。
var tokenTypeName = map[tokenType]string{
	tokenEOF: "输入结束", tokenNumber: "数字", tokenIdent: "标识符",
	tokenAnd: "and", tokenOr: "or", tokenNot: "not",
	tokenPlus: "+", tokenMinus: "-", tokenStar: "*", tokenSlash: "/",
	tokenLParen: "(", tokenRParen: ")", tokenComma: ",",
	tokenEq: "==", tokenNe: "!=", tokenGt: ">", tokenGe: ">=", tokenLt: "<", tokenLe: "<=",
	tokenIllegal: "非法字符",
}

// lexer 词法分析器，按字节顺序扫描源字符串生成token流。
type lexer struct {
	source string // 原始公式字符串，错误报告用
	pos    int    // 当前扫描字节偏移
}

// lex 将源字符串切分为token流，遇到非法字符立即返回错误并定位到字符位置。
func lex(source string) ([]token, error) {
	l := &lexer{source: source}
	// 容量按源串一半预分配，公式普遍短小，避免频繁扩容
	tokens := make([]token, 0, len(source)/2+1)
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.typ == tokenEOF {
			return tokens, nil
		}
	}
}

// next 扫描并返回下一个token；扫描完成返回tokenEOF，非法字符返回带位置错误。
func (l *lexer) next() (token, error) {
	l.skipWhitespace()
	if l.pos >= len(l.source) {
		return token{typ: tokenEOF, pos: l.pos}, nil
	}
	start := l.pos
	ch := l.source[l.pos]
	switch {
	case isDigit(ch):
		return l.lexNumber(start)
	case isIdentStart(ch):
		return l.lexIdent(start)
	}
	// 运算符与括号按字符逐一匹配，双字符运算符（== != >= <=）优先匹配
	switch ch {
	case '+':
		l.pos++
		return token{typ: tokenPlus, text: "+", pos: start}, nil
	case '-':
		l.pos++
		return token{typ: tokenMinus, text: "-", pos: start}, nil
	case '*':
		l.pos++
		return token{typ: tokenStar, text: "*", pos: start}, nil
	case '/':
		l.pos++
		return token{typ: tokenSlash, text: "/", pos: start}, nil
	case '(':
		l.pos++
		return token{typ: tokenLParen, text: "(", pos: start}, nil
	case ')':
		l.pos++
		return token{typ: tokenRParen, text: ")", pos: start}, nil
	case ',':
		l.pos++
		return token{typ: tokenComma, text: ",", pos: start}, nil
	case '=':
		if l.pos+1 < len(l.source) && l.source[l.pos+1] == '=' {
			l.pos += 2
			return token{typ: tokenEq, text: "==", pos: start}, nil
		}
		l.pos++
		return token{}, fmt.Errorf("公式 %q 第%d字符位置: 非法字符 %q，相等比较应使用 ==（不支持单个 =）", l.source, start+1, "=")
	case '!':
		if l.pos+1 < len(l.source) && l.source[l.pos+1] == '=' {
			l.pos += 2
			return token{typ: tokenNe, text: "!=", pos: start}, nil
		}
		l.pos++
		return token{}, fmt.Errorf("公式 %q 第%d字符位置: 非法字符 %q，不等比较应使用 !=", l.source, start+1, "!")
	case '>':
		if l.pos+1 < len(l.source) && l.source[l.pos+1] == '=' {
			l.pos += 2
			return token{typ: tokenGe, text: ">=", pos: start}, nil
		}
		l.pos++
		return token{typ: tokenGt, text: ">", pos: start}, nil
	case '<':
		if l.pos+1 < len(l.source) && l.source[l.pos+1] == '=' {
			l.pos += 2
			return token{typ: tokenLe, text: "<=", pos: start}, nil
		}
		l.pos++
		return token{typ: tokenLt, text: "<", pos: start}, nil
	default:
		l.pos++
		return token{}, fmt.Errorf("公式 %q 第%d字符位置: 非法字符 %q", l.source, start+1, string(ch))
	}
}

// skipWhitespace 跳过空格、制表符、换行等空白字符。
func (l *lexer) skipWhitespace() {
	for l.pos < len(l.source) {
		switch l.source[l.pos] {
		case ' ', '\t', '\n', '\r':
			l.pos++
		default:
			return
		}
	}
}

// lexNumber 扫描数字字面量：整数部分 + 可选小数部分（小数点后跟数字）。
func (l *lexer) lexNumber(start int) (token, error) {
	for l.pos < len(l.source) && isDigit(l.source[l.pos]) {
		l.pos++
	}
	// 小数部分：小数点后必须跟至少一位数字，否则按整数结束（如"1..2"的第二个点将作为非法字符报错）
	if l.pos < len(l.source) && l.source[l.pos] == '.' && l.pos+1 < len(l.source) && isDigit(l.source[l.pos+1]) {
		l.pos++
		for l.pos < len(l.source) && isDigit(l.source[l.pos]) {
			l.pos++
		}
	}
	text := l.source[start:l.pos]
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return token{}, fmt.Errorf("公式 %q 第%d字符位置: 数字 %q 格式错误", l.source, start+1, text)
	}
	return token{typ: tokenNumber, text: text, pos: start, value: value}, nil
}

// lexIdent 扫描标识符（变量名/函数名），并识别保留关键字and/or/not为逻辑运算符token。
func (l *lexer) lexIdent(start int) (token, error) {
	for l.pos < len(l.source) && isIdentPart(l.source[l.pos]) {
		l.pos++
	}
	text := l.source[start:l.pos]
	// 保留关键字（ADR-001 3.1 EBNF的逻辑运算符）单独成token，禁止再作为变量名/函数名使用
	switch text {
	case "and":
		return token{typ: tokenAnd, text: text, pos: start}, nil
	case "or":
		return token{typ: tokenOr, text: text, pos: start}, nil
	case "not":
		return token{typ: tokenNot, text: text, pos: start}, nil
	default:
		return token{typ: tokenIdent, text: text, pos: start}, nil
	}
}

// isDigit 判断字节是否为ASCII数字字符。
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// isIdentStart 判断字节是否为标识符起始字符（ASCII字母，ADR-001 3.1要求字母开头）。
func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// isIdentPart 判断字节是否为标识符组成部分（字母/数字/下划线）。
func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch) || ch == '_'
}
