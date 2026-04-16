package jql

import (
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// TokenType 词法单元类型
type TokenType string

const (
	TokenTypeField     TokenType = "field"
	TokenTypeOperator  TokenType = "operator"
	TokenTypeValue     TokenType = "value"
	TokenTypeFunction  TokenType = "function"
	TokenTypeLeftParen TokenType = "left_paren"
	TokenTypeRightParen TokenType = "right_paren"
	TokenTypeAnd       TokenType = "and"
	TokenTypeOr        TokenType = "or"
	TokenTypeNot       TokenType = "not"
)

// Token 词法单元
type Token struct {
	Type  TokenType
	Value string
}

// Parser JQL解析器
type Parser struct {
	tokens []Token
	pos    int
}

// NewParser 创建JQL解析器实例
func NewParser() *Parser {
	return &Parser{}
}

// Parse 解析JQL查询语句
func (p *Parser) Parse(query string) (bson.M, error) {
	// 词法分析
	tokens, err := p.tokenize(query)
	if err != nil {
		return nil, err
	}

	p.tokens = tokens
	p.pos = 0

	// 语法分析
	ast, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// 转换为MongoDB查询
	return p.convertToMongoQuery(ast), nil
}

// tokenize 词法分析
func (p *Parser) tokenize(query string) ([]Token, error) {
	var tokens []Token
	query = strings.TrimSpace(query)

	for len(query) > 0 {
		// 跳过空格
		if strings.HasPrefix(query, " ") {
			query = strings.TrimSpace(query)
			continue
		}

		// 处理函数
		if strings.Contains(query, "(") {
			// 查找函数名
			end := strings.Index(query, "(")
			// 检查前面是否是有效的函数名
			functionName := strings.TrimSpace(query[:end])
			if functionName != "" && isFieldName(functionName[0]) {
				tokens = append(tokens, Token{Type: TokenTypeFunction, Value: functionName})
				query = query[end:]
				continue
			}
		}

		// 处理括号
		if strings.HasPrefix(query, "(") {
			tokens = append(tokens, Token{Type: TokenTypeLeftParen, Value: "("})
			query = query[1:]
			continue
		}
		if strings.HasPrefix(query, ")") {
			tokens = append(tokens, Token{Type: TokenTypeRightParen, Value: ")"})
			query = query[1:]
			continue
		}

		// 处理逻辑操作符
		if strings.HasPrefix(query, "AND") || strings.HasPrefix(query, "and") {
			tokens = append(tokens, Token{Type: TokenTypeAnd, Value: "AND"})
			query = query[3:]
			continue
		}
		if strings.HasPrefix(query, "OR") || strings.HasPrefix(query, "or") {
			tokens = append(tokens, Token{Type: TokenTypeOr, Value: "OR"})
			query = query[2:]
			continue
		}
		if strings.HasPrefix(query, "NOT") || strings.HasPrefix(query, "not") {
			tokens = append(tokens, Token{Type: TokenTypeNot, Value: "NOT"})
			query = query[3:]
			continue
		}

		// 处理比较操作符
		operators := []string{">=", "<=", "!=", "=", ">", "<", "~"}
		found := false
		for _, op := range operators {
			if strings.HasPrefix(query, op) {
				tokens = append(tokens, Token{Type: TokenTypeOperator, Value: op})
				query = query[len(op):]
				found = true
				break
			}
		}
		if found {
			continue
		}

		// 处理字段名
		if isFieldName(query[0]) {
			end := 0
			for end < len(query) && (isFieldName(query[end]) || query[end] == '.') {
				end++
			}
			fieldName := strings.TrimSpace(query[:end])
			tokens = append(tokens, Token{Type: TokenTypeField, Value: fieldName})
			query = query[end:]
			continue
		}

		// 处理值
		if query[0] == '"' {
			// 带引号的值
			end := strings.Index(query[1:], "\"")
			if end == -1 {
				return nil, errors.New("unclosed string")
			}
			value := query[1 : end+1]
			tokens = append(tokens, Token{Type: TokenTypeValue, Value: value})
			query = query[end+2:]
			continue
		} else {
			// 不带引号的值
			end := 0
			for end < len(query) && !isWhitespace(query[end]) && !isOperator(query[end]) && query[end] != '(' && query[end] != ')' {
				end++
			}
			value := strings.TrimSpace(query[:end])
			tokens = append(tokens, Token{Type: TokenTypeValue, Value: value})
			query = query[end:]
			continue
		}

		return nil, errors.New("unexpected token: " + query[0:1])
	}

	return tokens, nil
}

// isFieldName 检查是否为字段名
func isFieldName(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// isWhitespace 检查是否为空白字符
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// isOperator 检查是否为操作符
func isOperator(c byte) bool {
	return c == '=' || c == '!' || c == '<' || c == '>' || c == '~'
}

// parseExpression 解析表达式
func (p *Parser) parseExpression() (interface{}, error) {
	return p.parseOrExpression()
}

// parseOrExpression 解析OR表达式
func (p *Parser) parseOrExpression() (interface{}, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return nil, err
	}

	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenTypeOr {
		p.pos++
		right, err := p.parseAndExpression()
		if err != nil {
			return nil, err
		}
		left = map[string]interface{}{"$or": []interface{}{left, right}}
	}

	return left, nil
}

// parseAndExpression 解析AND表达式
func (p *Parser) parseAndExpression() (interface{}, error) {
	left, err := p.parseNotExpression()
	if err != nil {
		return nil, err
	}

	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenTypeAnd {
		p.pos++
		right, err := p.parseNotExpression()
		if err != nil {
			return nil, err
		}
		left = map[string]interface{}{"$and": []interface{}{left, right}}
	}

	return left, nil
}

// parseNotExpression 解析NOT表达式
func (p *Parser) parseNotExpression() (interface{}, error) {
	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenTypeNot {
		p.pos++
		expr, err := p.parsePrimaryExpression()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"$not": expr}, nil
	}

	return p.parsePrimaryExpression()
}

// parsePrimaryExpression 解析基本表达式
func (p *Parser) parsePrimaryExpression() (interface{}, error) {
	if p.pos >= len(p.tokens) {
		return nil, errors.New("unexpected end of input")
	}

	token := p.tokens[p.pos]

	// 处理括号
	if token.Type == TokenTypeLeftParen {
		p.pos++
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenTypeRightParen {
			return nil, errors.New("expected closing parenthesis")
		}
		p.pos++
		return expr, nil
	}

	// 处理字段-操作符-值
	if token.Type == TokenTypeField {
		p.pos++
		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenTypeOperator {
			return nil, errors.New("expected operator")
		}

		operator := p.tokens[p.pos].Value
		p.pos++

		if p.pos >= len(p.tokens) || (p.tokens[p.pos].Type != TokenTypeValue && p.tokens[p.pos].Type != TokenTypeFunction) {
			return nil, errors.New("expected value or function")
		}

		var value interface{}
		if p.tokens[p.pos].Type == TokenTypeFunction {
			// 处理函数
			functionName := p.tokens[p.pos].Value
			p.pos++
			if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenTypeLeftParen {
				return nil, errors.New("expected opening parenthesis for function")
			}
			p.pos++
			// 处理函数参数（暂时不支持参数）
			if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenTypeRightParen {
				return nil, errors.New("expected closing parenthesis for function")
			}
			p.pos++

			// 处理内置函数
			switch strings.ToLower(functionName) {
			case "currentuser":
				value = "currentUser()" // 实际值由应用程序替换
			case "now":
				value = time.Now()
			case "startofday":
				value = time.Now().Truncate(24 * time.Hour)
			case "endofweek":
				// 计算本周结束时间
				now := time.Now()
				weekday := now.Weekday()
				if weekday == time.Sunday {
					weekday = 7
				}
				endOfWeek := now.AddDate(0, 0, 7-int(weekday))
				value = endOfWeek.Truncate(24 * time.Hour).Add(24*time.Hour - time.Second)
			default:
				return nil, errors.New("unknown function: " + functionName)
			}
		} else {
			// 处理普通值
			value = p.tokens[p.pos].Value
			p.pos++
		}

		// 转换为MongoDB查询
		return p.convertCondition(token.Value, operator, value), nil
	}

	return nil, errors.New("unexpected token: " + token.Value)
}

// convertCondition 转换条件为MongoDB查询
func (p *Parser) convertCondition(field, operator string, value interface{}) bson.M {
	switch operator {
	case "=":
		return bson.M{field: value}
	case "!=":
		return bson.M{field: bson.M{"$ne": value}}
	case ">":
		return bson.M{field: bson.M{"$gt": value}}
	case "<":
		return bson.M{field: bson.M{"$lt": value}}
	case ">=":
		return bson.M{field: bson.M{"$gte": value}}
	case "<=":
		return bson.M{field: bson.M{"$lte": value}}
	case "~":
		return bson.M{field: bson.M{"$regex": value, "$options": "i"}}
	default:
		return bson.M{field: value}
	}
}

// convertToMongoQuery 转换AST为MongoDB查询
func (p *Parser) convertToMongoQuery(ast interface{}) bson.M {
	switch v := ast.(type) {
	case map[string]interface{}:
		result := bson.M{}
		for key, val := range v {
			switch key {
			case "$and", "$or":
				var conditions []bson.M
				for _, item := range val.([]interface{}) {
					conditions = append(conditions, p.convertToMongoQuery(item))
				}
				result[key] = conditions
			case "$not":
				result[key] = p.convertToMongoQuery(val)
			default:
				result[key] = val
			}
		}
		return result
	default:
		return bson.M{}
	}
}

// ParseQuery 解析JQL查询语句并返回MongoDB查询
func ParseQuery(query string) (bson.M, error) {
	parser := NewParser()
	return parser.Parse(query)
}
