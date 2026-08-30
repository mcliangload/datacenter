package dql

import (
	"fmt"
	"math"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"datacenter/internal/model"
)

// BuildFilter 将 DQL AST 转换为 MongoDB 过滤条件。
// schemas: 集合ID(hex) → 标签定义，用于类型校验与值规范化；
// collection 特殊字段由调用方处理后在此跳过（返回 nil 不参与过滤）。
func BuildFilter(node Node, schemas map[string][]model.TagDefinition) (bson.M, error) {
	return buildExpr(node, schemas)
}

func buildExpr(n Node, schemas map[string][]model.TagDefinition) (bson.M, error) {
	switch v := n.(type) {
	case *And:
		l, err := buildExpr(v.Left, schemas)
		if err != nil {
			return nil, err
		}
		r, err := buildExpr(v.Right, schemas)
		if err != nil {
			return nil, err
		}
		if l == nil {
			return r, nil
		}
		if r == nil {
			return l, nil
		}
		return bson.M{"$and": []bson.M{l, r}}, nil
	case *Or:
		l, err := buildExpr(v.Left, schemas)
		if err != nil {
			return nil, err
		}
		r, err := buildExpr(v.Right, schemas)
		if err != nil {
			return nil, err
		}
		if l == nil {
			return r, nil
		}
		if r == nil {
			return l, nil
		}
		return bson.M{"$or": []bson.M{l, r}}, nil
	case *Cond:
		if v.Field == "collection" || v.Field == "parent" || v.Field == "ancestor" {
			return nil, nil // 特殊字段：由服务层解析（集合范围 / 关联关系）
		}
		return buildCond(v, schemas)
	}
	return nil, fmt.Errorf("未知表达式节点")
}

func buildCond(c *Cond, schemas map[string][]model.TagDefinition) (bson.M, error) {
	types := fieldTypes(schemas, c.Field)
	if len(types) == 0 {
		return nil, fmt.Errorf("标签不存在: %s", c.Field)
	}
	field := "tags." + c.Field

	switch c.Op {
	case "=", "!=":
		v, err := coerceScalar(c.Field, c.Value, types)
		if err != nil {
			return nil, err
		}
		if c.Op == "=" {
			return bson.M{field: v}, nil
		}
		return bson.M{field: bson.M{"$ne": v}}, nil

	case ">", ">=", "<", "<=":
		if !allNumericOrDate(types) {
			return nil, fmt.Errorf("标签 %s 不支持范围查询（仅 int/float/date 类型）", c.Field)
		}
		v, err := coerceScalar(c.Field, c.Value, types)
		if err != nil {
			return nil, err
		}
		mongoOp := map[string]string{">": "$gt", ">=": "$gte", "<": "$lt", "<=": "$lte"}[c.Op]
		return bson.M{field: bson.M{mongoOp: v}}, nil

	case "IN":
		vals, ok := c.Value.([]interface{})
		if !ok {
			return nil, fmt.Errorf("IN 需要值列表")
		}
		arr := make([]interface{}, 0, len(vals))
		for _, item := range vals {
			v, err := coerceScalar(c.Field, item, types)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return bson.M{field: bson.M{"$in": arr}}, nil

	case "LIKE":
		if !allStringLike(types) {
			return nil, fmt.Errorf("标签 %s 不支持 LIKE（仅 string/enum 类型）", c.Field)
		}
		s, ok := c.Value.(string)
		if !ok {
			return nil, fmt.Errorf("LIKE 需要字符串值")
		}
		// 包含匹配（大小写不敏感），特殊字符转义
		return bson.M{field: bson.M{"$regex": regexp.QuoteMeta(s), "$options": "i"}}, nil

	case "EXISTS":
		b, ok := c.Value.(bool)
		if !ok {
			return nil, fmt.Errorf("EXISTS 需要 true/false")
		}
		return bson.M{field: bson.M{"$exists": b}}, nil
	}
	return nil, fmt.Errorf("不支持的运算符: %s", c.Op)
}

// fieldTypes 收集字段在目标集合中的全部类型定义
func fieldTypes(schemas map[string][]model.TagDefinition, field string) []model.TagType {
	var out []model.TagType
	for _, schema := range schemas {
		for _, t := range schema {
			if t.Name == field {
				out = append(out, t.Type)
			}
		}
	}
	return out
}

func uniqueTypes(types []model.TagType) []model.TagType {
	seen := make(map[model.TagType]bool, len(types))
	var out []model.TagType
	for _, t := range types {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func allNumericOrDate(types []model.TagType) bool {
	for _, t := range uniqueTypes(types) {
		switch t {
		case model.TagTypeInt, model.TagTypeFloat, model.TagTypeDate:
		default:
			return false
		}
	}
	return true
}

func allStringLike(types []model.TagType) bool {
	for _, t := range uniqueTypes(types) {
		switch t {
		case model.TagTypeString, model.TagTypeEnum:
		default:
			return false
		}
	}
	return true
}

// coerceScalar 按字段类型规范化查询值。
// 所有目标集合类型一致时按该类型严格校验；跨集合类型不一致时宽松转换（数值转 float64，
// MongoDB 数值跨类型比较按值相等，可同时命中 int/float）。
func coerceScalar(field string, v interface{}, types []model.TagType) (interface{}, error) {
	uniq := uniqueTypes(types)
	if len(uniq) == 1 {
		switch uniq[0] {
		case model.TagTypeInt:
			if n, ok := toInt64(v); ok {
				return n, nil
			}
			return nil, fmt.Errorf("标签 %s 应为整数，得到 %v", field, v)
		case model.TagTypeFloat:
			if f, ok := toFloat64(v); ok {
				return f, nil
			}
			return nil, fmt.Errorf("标签 %s 应为数值，得到 %v", field, v)
		case model.TagTypeBool:
			if b, ok := v.(bool); ok {
				return b, nil
			}
			return nil, fmt.Errorf("标签 %s 应为 true/false，得到 %v", field, v)
		case model.TagTypeDate:
			if s, ok := v.(string); ok {
				if tm, err := parseDate(s); err == nil {
					return tm, nil
				}
			}
			return nil, fmt.Errorf("标签 %s 应为日期（RFC3339 或 YYYY-MM-DD）", field)
		case model.TagTypeString, model.TagTypeEnum:
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("标签 %s 应为字符串，得到 %v", field, v)
			}
			return s, nil
		}
	}
	// 混合类型/未知：宽松转换
	switch val := v.(type) {
	case string:
		if tm, err := parseDate(val); err == nil {
			return tm, nil
		}
		return val, nil
	case bool:
		return val, nil
	case int, int32, int64, float64:
		f, ok := toFloat64(val)
		if !ok {
			return nil, fmt.Errorf("标签 %s 的值类型不支持: %T", field, v)
		}
		return f, nil
	default:
		return nil, fmt.Errorf("标签 %s 的值类型不支持: %T", field, v)
	}
}

func parseDate(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if tm, err := time.Parse(layout, s); err == nil {
			return tm, nil
		}
	}
	return time.Time{}, fmt.Errorf("非法日期: %s", s)
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		if n == math.Trunc(n) && n >= math.MinInt64 && n <= math.MaxInt64 {
			return int64(n), true
		}
	}
	return 0, false
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// CollectionRef 语句中的集合限定条件
type CollectionRef struct {
	Op    string // = != IN
	Names []string
}

// ExtractCollections 提取语句中的 collection 条件并校验运算符合法性
func ExtractCollections(n Node) ([]CollectionRef, error) {
	var refs []CollectionRef
	walk(n, func(c *Cond) {
		if c.Field != "collection" {
			return
		}
		refs = append(refs, CollectionRef{Op: c.Op, Names: collectionNames(c)})
	})
	for _, r := range refs {
		switch r.Op {
		case "=", "!=":
			if len(r.Names) != 1 {
				return nil, fmt.Errorf("collection %s 需要单个集合名", r.Op)
			}
		case "IN":
			if len(r.Names) == 0 {
				return nil, fmt.Errorf("collection IN 需要至少一个集合名")
			}
		default:
			return nil, fmt.Errorf("collection 字段仅支持 =、!=、IN，不支持 %s", r.Op)
		}
	}
	return refs, nil
}

func collectionNames(c *Cond) []string {
	if vals, ok := c.Value.([]interface{}); ok {
		names := make([]string, 0, len(vals))
		for _, v := range vals {
			if s, ok := v.(string); ok {
				names = append(names, s)
			}
		}
		return names
	}
	if s, ok := c.Value.(string); ok {
		return []string{s}
	}
	return nil
}

func walk(n Node, fn func(*Cond)) {
	switch v := n.(type) {
	case *Cond:
		fn(v)
	case *And:
		walk(v.Left, fn)
		walk(v.Right, fn)
	case *Or:
		walk(v.Left, fn)
		walk(v.Right, fn)
	}
}

// RelationRef 语句中的关联关系限定（v2：parent 直接子 / ancestor 子树）
type RelationRef struct {
	Field  string   // parent | ancestor
	Op     string   // = | IN
	Values []string // 数据项 id（hex）
}

// ExtractRelationRefs 提取语句中的 parent/ancestor 条件并校验
func ExtractRelationRefs(n Node) ([]RelationRef, error) {
	var refs []RelationRef
	walk(n, func(c *Cond) {
		if c.Field != "parent" && c.Field != "ancestor" {
			return
		}
		refs = append(refs, RelationRef{Field: c.Field, Op: c.Op, Values: collectionNames(c)})
	})
	for _, r := range refs {
		switch r.Op {
		case "=", "IN":
			if len(r.Values) == 0 {
				return nil, fmt.Errorf("%s 需要至少一个数据项 id", r.Field)
			}
		default:
			return nil, fmt.Errorf("%s 字段仅支持 =、IN，不支持 %s", r.Field, r.Op)
		}
	}
	return refs, nil
}

// 供外部使用的辅助：判断语句是否引用了 collection 字段
func hasCollectionRef(refs []CollectionRef) bool {
	return len(refs) > 0
}
