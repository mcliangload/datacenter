package jql

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type TokenType string

const (
	TokenTypeField      TokenType = "field"
	TokenTypeOperator   TokenType = "operator"
	TokenTypeValue      TokenType = "value"
	TokenTypeFunction   TokenType = "function"
	TokenTypeLeftParen  TokenType = "left_paren"
	TokenTypeRightParen TokenType = "right_paren"
	TokenTypeAnd        TokenType = "and"
	TokenTypeOr         TokenType = "or"
	TokenTypeNot        TokenType = "not"
	TokenTypeComma      TokenType = "comma"
	TokenTypeIn         TokenType = "in"
	TokenTypeNotIn      TokenType = "not_in"
	TokenTypeLike       TokenType = "like"
	TokenTypeIsNull     TokenType = "is_null"
	TokenTypeIsNotNull  TokenType = "is_not_null"
)

type Token struct {
	Type  TokenType
	Value string
}

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	tokens, err := p.tokenize(query)
	if err != nil {
		return nil, err
	}

	p.tokens = tokens
	p.pos = 0

	ast, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return p.convertToMongoQuery(ast), nil
}

func (p *Parser) tokenize(query string) ([]Token, error) {
	var tokens []Token
	query = strings.TrimSpace(query)

	for len(query) > 0 {
		if strings.HasPrefix(query, " ") {
			query = strings.TrimSpace(query)
			continue
		}

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
		if strings.HasPrefix(query, ",") {
			tokens = append(tokens, Token{Type: TokenTypeComma, Value: ","})
			query = query[1:]
			continue
		}

		lowerQuery := strings.ToLower(query)
		if strings.HasPrefix(lowerQuery, "and") && (len(query) == 3 || !isAlphanumeric(byte(query[3]))) {
			tokens = append(tokens, Token{Type: TokenTypeAnd, Value: "AND"})
			query = query[3:]
			continue
		}
		if strings.HasPrefix(lowerQuery, "or") && (len(query) == 2 || !isAlphanumeric(byte(query[2]))) {
			tokens = append(tokens, Token{Type: TokenTypeOr, Value: "OR"})
			query = query[2:]
			continue
		}
		if strings.HasPrefix(lowerQuery, "not") && (len(query) == 3 || !isAlphanumeric(byte(query[3]))) {
			tokens = append(tokens, Token{Type: TokenTypeNot, Value: "NOT"})
			query = query[3:]
			continue
		}

		if strings.HasPrefix(lowerQuery, "is null") {
			tokens = append(tokens, Token{Type: TokenTypeIsNull, Value: "IS NULL"})
			query = query[7:]
			continue
		}
		if strings.HasPrefix(lowerQuery, "is not null") {
			tokens = append(tokens, Token{Type: TokenTypeIsNotNull, Value: "IS NOT NULL"})
			query = query[11:]
			continue
		}

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

		if isFieldName(query[0]) {
			end := 0
			for end < len(query) && (isFieldName(query[end]) || query[end] == '.' || query[end] == '_') {
				end++
			}
			fieldName := strings.TrimSpace(query[:end])
			tokens = append(tokens, Token{Type: TokenTypeField, Value: fieldName})
			query = query[end:]
			continue
		}

		if query[0] == '\'' || query[0] == '"' {
			quote := query[0]
			end := strings.Index(query[1:], string(quote))
			if end == -1 {
				return nil, errors.New("unclosed string")
			}
			value := query[1 : end+1]
			tokens = append(tokens, Token{Type: TokenTypeValue, Value: value})
			query = query[end+2:]
			continue
		} else {
			end := 0
			for end < len(query) && !isWhitespace(query[end]) && query[end] != '(' && query[end] != ')' && query[end] != ',' {
				end++
			}
			value := strings.TrimSpace(query[:end])
			if value != "" {
				if strings.ToLower(value) == "in" {
					tokens = append(tokens, Token{Type: TokenTypeIn, Value: "IN"})
				} else if strings.ToLower(value) == "not" {
					nextQuery := strings.TrimSpace(query[3:])
					if strings.HasPrefix(strings.ToLower(nextQuery), "in") {
						tokens = append(tokens, Token{Type: TokenTypeNotIn, Value: "NOT IN"})
						query = strings.TrimSpace(query[3:])
						continue
					}
					tokens = append(tokens, Token{Type: TokenTypeValue, Value: value})
				} else {
					tokens = append(tokens, Token{Type: TokenTypeValue, Value: value})
				}
			}
			query = query[end:]
			continue
		}
	}

	return tokens, nil
}

func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isFieldName(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.'
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func (p *Parser) parseExpression() (interface{}, error) {
	return p.parseOrExpression()
}

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

func (p *Parser) parsePrimaryExpression() (interface{}, error) {
	if p.pos >= len(p.tokens) {
		return nil, errors.New("unexpected end of input")
	}

	token := p.tokens[p.pos]

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

	if token.Type == TokenTypeField {
		p.pos++
		fieldName := token.Value

		if p.pos >= len(p.tokens) {
			return nil, errors.New("expected operator after field")
		}

		opToken := p.tokens[p.pos]

		if opToken.Type == TokenTypeIsNull {
			p.pos++
			return p.convertCondition(fieldName, "IS NULL", nil), nil
		}

		if opToken.Type == TokenTypeIsNotNull {
			p.pos++
			return p.convertCondition(fieldName, "IS NOT NULL", nil), nil
		}

		if p.pos >= len(p.tokens) || opToken.Type != TokenTypeOperator {
			return nil, errors.New("expected operator")
		}

		operator := opToken.Value
		p.pos++

		if p.pos >= len(p.tokens) {
			return nil, errors.New("expected value after operator")
		}

		valueToken := p.tokens[p.pos]

		if valueToken.Type == TokenTypeIn || valueToken.Type == TokenTypeNotIn {
			isNotIn := valueToken.Type == TokenTypeNotIn
			p.pos++

			if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenTypeLeftParen {
				return nil, errors.New("expected ( after IN/NOT IN")
			}
			p.pos++

			var values []interface{}
			for {
				if p.pos >= len(p.tokens) {
					return nil, errors.New("expected closing parenthesis")
				}

				if p.tokens[p.pos].Type == TokenTypeRightParen {
					p.pos++
					break
				}

				if p.tokens[p.pos].Type == TokenTypeComma {
					p.pos++
					continue
				}

				v := p.parseValue()
				values = append(values, v)

				if p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenTypeComma {
					p.pos++
				}
			}

			if isNotIn {
				return p.convertCondition(fieldName, "NOT IN", values), nil
			}
			return p.convertCondition(fieldName, "IN", values), nil
		}

		var value interface{}
		if valueToken.Type == TokenTypeFunction {
			value = p.parseFunction(valueToken.Value)
		} else {
			value = p.parseValue()
		}

		p.pos++

		return p.convertCondition(fieldName, operator, value), nil
	}

	return nil, fmt.Errorf("unexpected token: %s", token.Value)
}

func (p *Parser) parseValue() interface{} {
	if p.pos >= len(p.tokens) {
		return nil
	}

	token := p.tokens[p.pos]

	switch token.Type {
	case TokenTypeValue:
		return p.parseValueType(token.Value)
	case TokenTypeFunction:
		return p.parseFunction(token.Value)
	default:
		return token.Value
	}
}

func (p *Parser) parseValueType(value string) interface{} {
	if matched, _ := regexp.MatchString(`^-?\d+$`, value); matched {
		intVal := 0
		fmt.Sscanf(value, "%d", &intVal)
		return intVal
	}

	if matched, _ := regexp.MatchString(`^-?\d+\.\d+$`, value); matched {
		var floatVal float64
		fmt.Sscanf(value, "%f", &floatVal)
		return floatVal
	}

	lower := strings.ToLower(value)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}

	return value
}

func (p *Parser) parseFunction(funcName string) interface{} {
	switch strings.ToLower(funcName) {
	case "currentuser":
		return "currentUser()"
	case "now":
		return time.Now()
	case "startofday":
		return time.Now().Truncate(24 * time.Hour)
	case "endofday":
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	case "startofweek":
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return now.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)
	case "endofweek":
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		endOfWeek := now.AddDate(0, 0, 7-weekday)
		return time.Date(endOfWeek.Year(), endOfWeek.Month(), endOfWeek.Day(), 23, 59, 59, 999999999, endOfWeek.Location())
	case "startofmonth":
		now := time.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "endofmonth":
		now := time.Now()
		return time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 999999999, now.Location())
	}
	return nil
}

func (p *Parser) convertCondition(field, operator string, value interface{}) bson.M {
	switch operator {
	case "=":
		if value == nil {
			return bson.M{field: bson.M{"$exists": false}}
		}
		return bson.M{field: value}
	case "!=":
		if value == nil {
			return bson.M{field: bson.M{"$exists": true}}
		}
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
	case "IN":
		return bson.M{field: bson.M{"$in": value}}
	case "NOT IN":
		return bson.M{field: bson.M{"$nin": value}}
	case "IS NULL":
		return bson.M{field: bson.M{"$exists": false}}
	case "IS NOT NULL":
		return bson.M{field: bson.M{"$exists": true}}
	default:
		if value == nil {
			return bson.M{field: bson.M{"$exists": false}}
		}
		return bson.M{field: value}
	}
}

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

func ParseQuery(query string) (bson.M, error) {
	parser := NewParser()
	return parser.Parse(query)
}

func GetExampleQueries() []string {
	return []string{
		`status = "active"`,
		`name contains "产品"`,
		`price > 100`,
		`status IN ("active", "pending")`,
		`category NOT IN ("deleted", "archived")`,
		`title LIKE "重要%"`,
		`created > "2024-01-01"`,
		`assignee IS NULL`,
		`email IS NOT NULL`,
		`status = "active" AND price > 100`,
		`name = "A" OR name = "B"`,
		`(status = "active") AND (price > 100 OR price < 50)`,
		`created > StartOfWeek() AND module = "movie"`,
		`updated < EndOfMonth() AND status NOT IN ("deleted", "archived")`,
	}
}

func ValidateJQL(query string) error {
	_, err := ParseQuery(query)
	return err
}
