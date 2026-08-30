package dql

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// tokenType 词法单元类型
type tokenType int

const (
	tokEOF tokenType = iota
	tokIdent
	tokString
	tokNumber
	tokBool
	tokLParen
	tokRParen
	tokComma
	tokOp // = != <> > >= < <=
	tokAnd
	tokOr
	tokIn
	tokExists
	tokLike
)

type token struct {
	typ tokenType
	lit string
	pos int
}

type lexer struct {
	src string
	pos int
}

// lex 将 DQL 语句切分为词法单元
func lex(src string) ([]token, error) {
	l := &lexer{src: src}
	var toks []token
	for {
		t, err := l.next()
		if err != nil {
			return nil, err
		}
		toks = append(toks, t)
		if t.typ == tokEOF {
			return toks, nil
		}
	}
}

func (l *lexer) next() (token, error) {
	for l.pos < len(l.src) {
		r, size := utf8.DecodeRuneInString(l.src[l.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		l.pos += size
	}
	if l.pos >= len(l.src) {
		return token{typ: tokEOF, pos: l.pos}, nil
	}
	start := l.pos
	c := l.src[l.pos]
	switch c {
	case '(':
		l.pos++
		return token{tokLParen, "(", start}, nil
	case ')':
		l.pos++
		return token{tokRParen, ")", start}, nil
	case ',':
		l.pos++
		return token{tokComma, ",", start}, nil
	case '=':
		l.pos++
		return token{tokOp, "=", start}, nil
	case '>':
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == '=' {
			l.pos += 2
			return token{tokOp, ">=", start}, nil
		}
		l.pos++
		return token{tokOp, ">", start}, nil
	case '<':
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == '=' {
			l.pos += 2
			return token{tokOp, "<=", start}, nil
		}
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == '>' {
			l.pos += 2
			return token{tokOp, "!=", start}, nil
		}
		l.pos++
		return token{tokOp, "<", start}, nil
	case '!':
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == '=' {
			l.pos += 2
			return token{tokOp, "!=", start}, nil
		}
		return token{}, fmt.Errorf("位置 %d: 非法字符 '!'（是否想用 != ？）", start)
	case '\'', '"':
		return l.lexString(c)
	}
	if c == '-' || (c >= '0' && c <= '9') {
		return l.lexNumber()
	}
	if isIdentStart(rune(c)) {
		return l.lexWord()
	}
	return token{}, fmt.Errorf("位置 %d: 无法识别的字符 %q", start, c)
}

// lexString 读取带引号字符串，支持 \\ \' \" \n \t 转义
func (l *lexer) lexString(quote byte) (token, error) {
	start := l.pos
	l.pos++ // 跳过开引号
	var sb strings.Builder
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\\' && l.pos+1 < len(l.src) {
			switch n := l.src[l.pos+1]; n {
			case '\\', '\'', '"':
				sb.WriteByte(n)
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			default:
				sb.WriteByte(n)
			}
			l.pos += 2
			continue
		}
		if c == quote {
			l.pos++
			return token{tokString, sb.String(), start}, nil
		}
		sb.WriteByte(c)
		l.pos++
	}
	return token{}, fmt.Errorf("位置 %d: 字符串未闭合", start)
}

func (l *lexer) lexNumber() (token, error) {
	start := l.pos
	if l.src[l.pos] == '-' {
		l.pos++
	}
	for l.pos < len(l.src) && l.src[l.pos] >= '0' && l.src[l.pos] <= '9' {
		l.pos++
	}
	if l.pos < len(l.src) && l.src[l.pos] == '.' {
		l.pos++
		for l.pos < len(l.src) && l.src[l.pos] >= '0' && l.src[l.pos] <= '9' {
			l.pos++
		}
	}
	return token{tokNumber, l.src[start:l.pos], start}, nil
}

func (l *lexer) lexWord() (token, error) {
	start := l.pos
	for l.pos < len(l.src) {
		r, size := utf8.DecodeRuneInString(l.src[l.pos:])
		if !isIdentPart(r) {
			break
		}
		l.pos += size
	}
	word := l.src[start:l.pos]
	switch strings.ToUpper(word) {
	case "AND":
		return token{tokAnd, word, start}, nil
	case "OR":
		return token{tokOr, word, start}, nil
	case "IN":
		return token{tokIn, word, start}, nil
	case "EXISTS":
		return token{tokExists, word, start}, nil
	case "LIKE":
		return token{tokLike, word, start}, nil
	case "TRUE", "FALSE":
		return token{tokBool, strings.ToLower(word), start}, nil
	}
	return token{tokIdent, word, start}, nil
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
