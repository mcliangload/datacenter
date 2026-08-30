package dql

import (
	"fmt"
	"strconv"
	"strings"
)

// 语法（AND 优先级高于 OR，支持括号）：
//
//	expr     := orExpr
//	orExpr   := andExpr (OR andExpr)*
//	andExpr  := primary (AND primary)*
//	primary  := '(' expr ')' | cond
//	cond     := field op valueList
//	field    := IDENT | STRING
//	op       := '=' | '!=' | '>' | '>=' | '<' | '<=' | IN | EXISTS | LIKE
//	value    := STRING | NUMBER | BOOL | IDENT(裸字符串)
type parser struct {
	toks []token
	pos  int
}

// Parse 解析 DQL 语句为 AST
func Parse(src string) (Node, error) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	if p.peek().typ == tokEOF {
		return nil, fmt.Errorf("语句为空")
	}
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().typ != tokEOF {
		return nil, fmt.Errorf("位置 %d: 多余的输入 %q", p.peek().pos, p.peek().lit)
	}
	return node, nil
}

func (p *parser) peek() token { return p.toks[p.pos] }

func (p *parser) advance() token {
	t := p.toks[p.pos]
	if t.typ != tokEOF {
		p.pos++
	}
	return t
}

func (p *parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().typ == tokOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &Or{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Node, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for p.peek().typ == tokAnd {
		p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = &And{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parsePrimary() (Node, error) {
	if p.peek().typ == tokLParen {
		p.advance()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().typ != tokRParen {
			return nil, fmt.Errorf("位置 %d: 缺少右括号 )", p.peek().pos)
		}
		p.advance()
		return inner, nil
	}
	return p.parseCond()
}

func (p *parser) parseCond() (Node, error) {
	fieldTok := p.peek()
	if fieldTok.typ != tokIdent && fieldTok.typ != tokString {
		return nil, fmt.Errorf("位置 %d: 期望字段名，得到 %q", fieldTok.pos, fieldTok.lit)
	}
	p.advance()

	opTok := p.peek()
	switch opTok.typ {
	case tokOp, tokIn, tokExists, tokLike:
	default:
		return nil, fmt.Errorf("位置 %d: 期望运算符（=、!=、>、>=、<、<=、IN、EXISTS、LIKE），得到 %q", opTok.pos, opTok.lit)
	}
	p.advance()

	cond := &Cond{Field: fieldTok.lit, Op: opTok.lit}
	switch opTok.typ {
	case tokIn:
		// 兼容 IN (a, b) 与 IN a, b 两种写法
		if p.peek().typ == tokLParen {
			p.advance()
		}
		vals, err := p.parseValueList()
		if err != nil {
			return nil, err
		}
		if len(vals) == 0 {
			return nil, fmt.Errorf("位置 %d: IN 需要至少一个值", opTok.pos)
		}
		cond.Value = vals
		if p.peek().typ == tokRParen {
			p.advance()
		}
	case tokExists:
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("位置 %d: EXISTS 需要 true/false", opTok.pos)
		}
		cond.Value = b
	default:
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		cond.Value = v
	}
	return cond, nil
}

func (p *parser) parseValueList() ([]interface{}, error) {
	var vals []interface{}
	for {
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
		if p.peek().typ != tokComma {
			break
		}
		p.advance()
	}
	return vals, nil
}

func (p *parser) parseValue() (interface{}, error) {
	t := p.peek()
	switch t.typ {
	case tokString, tokNumber, tokBool:
		p.advance()
		return parseLiteral(t)
	case tokIdent:
		// 兼容裸值写法：name = demo 等价于 name = "demo"
		p.advance()
		return t.lit, nil
	default:
		return nil, fmt.Errorf("位置 %d: 期望值（字符串/数字/true/false），得到 %q", t.pos, t.lit)
	}
}

func parseLiteral(t token) (interface{}, error) {
	switch t.typ {
	case tokString:
		return t.lit, nil
	case tokBool:
		return t.lit == "true", nil
	case tokNumber:
		if strings.Contains(t.lit, ".") {
			f, err := strconv.ParseFloat(t.lit, 64)
			if err != nil {
				return nil, fmt.Errorf("位置 %d: 非法数字 %q", t.pos, t.lit)
			}
			return f, nil
		}
		n, err := strconv.ParseInt(t.lit, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("位置 %d: 非法数字 %q", t.pos, t.lit)
		}
		return n, nil
	}
	return nil, fmt.Errorf("非法字面量 %q", t.lit)
}
