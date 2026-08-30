package service

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"datacenter/internal/errno"
	"datacenter/internal/model"
)

// 系统保留字段：标签名不得与之冲突（需求分解 §5.3 / Q11）
var reservedTagNames = map[string]struct{}{
	"id": {}, "path": {}, "collection_id": {}, "tags": {},
	"tag_source": {}, "scrape_status": {}, "last_scraped_at": {},
	"created_by": {}, "created_at": {}, "updated_at": {},
}

func tagSchemaErr(msg string) *errno.Error {
	return errno.New(errno.ErrTagSchemaInvalid.Code, errno.ErrTagSchemaInvalid.HTTPStatus, "标签定义不合法: "+msg)
}

func tagValueErr(msg string) *errno.Error {
	return errno.New(errno.ErrTagValueInvalid.Code, errno.ErrTagValueInvalid.HTTPStatus, "标签值不合法: "+msg)
}

// ValidateTagSchema 校验标签定义集合：
// 名称唯一、不冲突保留字段、类型合法、enum/array/object 附加约束正确。
func ValidateTagSchema(schema []model.TagDefinition) *errno.Error {
	seen := make(map[string]bool, len(schema))
	for _, t := range schema {
		if strings.TrimSpace(t.Name) == "" {
			return tagSchemaErr("标签名不能为空")
		}
		if _, reserved := reservedTagNames[t.Name]; reserved {
			return tagSchemaErr("标签名与系统保留字段冲突: " + t.Name)
		}
		if seen[t.Name] {
			return tagSchemaErr("标签名重复: " + t.Name)
		}
		seen[t.Name] = true
		if err := validateTagType(t); err != nil {
			return err
		}
	}
	return nil
}

func validateTagType(t model.TagDefinition) *errno.Error {
	switch t.Type {
	case model.TagTypeString, model.TagTypeInt, model.TagTypeFloat, model.TagTypeBool, model.TagTypeDate:
		return nil
	case model.TagTypeEnum:
		if len(t.EnumValues) == 0 {
			return tagSchemaErr("enum 标签 " + t.Name + " 必须配置 enum_values")
		}
		return nil
	case model.TagTypeArray:
		if !isBaseTagType(t.ElementType) {
			return tagSchemaErr("array 标签 " + t.Name + " 的 element_type 必须是基础类型")
		}
		return nil
	case model.TagTypeObject:
		if len(t.Fields) == 0 {
			return tagSchemaErr("object 标签 " + t.Name + " 必须配置 fields")
		}
		seen := make(map[string]bool, len(t.Fields))
		for _, f := range t.Fields {
			if strings.TrimSpace(f.Name) == "" {
				return tagSchemaErr("object 标签 " + t.Name + " 的子字段名不能为空")
			}
			if seen[f.Name] {
				return tagSchemaErr("object 标签 " + t.Name + " 子字段名重复: " + f.Name)
			}
			seen[f.Name] = true
			if err := validateTagType(f); err != nil {
				return err
			}
		}
		return nil
	default:
		return tagSchemaErr("不支持的标签类型: " + string(t.Type))
	}
}

func isBaseTagType(t model.TagType) bool {
	switch t {
	case model.TagTypeString, model.TagTypeInt, model.TagTypeFloat, model.TagTypeBool, model.TagTypeDate:
		return true
	}
	return false
}

// ValidateAndNormalizeTags 校验并规范化标签值：
//   - 必填标签必须存在；
//   - 各标签按定义做类型/枚举/数组/嵌套对象校验；
//   - allowUnknown=true 时（刮削场景）忽略未知标签，否则（手动场景）拒绝。
//
// 返回规范化后的标签集合（int 统一为 int64、float 统一为 float64、date 统一为 time.Time）。
func ValidateAndNormalizeTags(schema []model.TagDefinition, tags map[string]interface{}, allowUnknown bool) (map[string]interface{}, *errno.Error) {
	if tags == nil {
		tags = map[string]interface{}{}
	}
	schemaByName := make(map[string]model.TagDefinition, len(schema))
	for _, t := range schema {
		schemaByName[t.Name] = t
	}

	// 必填检查
	for _, t := range schema {
		if t.Required {
			if _, ok := tags[t.Name]; !ok {
				return nil, tagValueErr("缺少必填标签: " + t.Name)
			}
		}
	}

	out := make(map[string]interface{}, len(tags))
	for k, v := range tags {
		t, ok := schemaByName[k]
		if !ok {
			if allowUnknown {
				continue
			}
			return nil, tagValueErr("未知标签: " + k)
		}
		nv, err := normalizeTagValue(t, v)
		if err != nil {
			return nil, err
		}
		out[k] = nv
	}
	return out, nil
}

func normalizeTagValue(t model.TagDefinition, v interface{}) (interface{}, *errno.Error) {
	switch t.Type {
	case model.TagTypeString:
		s, ok := v.(string)
		if !ok {
			return nil, tagValueErr("标签 " + t.Name + " 应为 string")
		}
		return s, nil

	case model.TagTypeInt:
		n, ok := toInt64(v)
		if !ok {
			return nil, tagValueErr("标签 " + t.Name + " 应为 int")
		}
		return n, nil

	case model.TagTypeFloat:
		f, ok := toFloat64(v)
		if !ok {
			return nil, tagValueErr("标签 " + t.Name + " 应为 float")
		}
		return f, nil

	case model.TagTypeBool:
		b, ok := v.(bool)
		if !ok {
			return nil, tagValueErr("标签 " + t.Name + " 应为 bool")
		}
		return b, nil

	case model.TagTypeDate:
		switch d := v.(type) {
		case time.Time:
			return d, nil
		case string:
			for _, layout := range []string{time.RFC3339, "2006-01-02"} {
				if tm, err := time.Parse(layout, d); err == nil {
					return tm, nil
				}
			}
			return nil, tagValueErr("标签 " + t.Name + " 应为日期/时间（RFC3339 或 YYYY-MM-DD）")
		default:
			return nil, tagValueErr("标签 " + t.Name + " 应为日期/时间")
		}

	case model.TagTypeEnum:
		s, ok := v.(string)
		if !ok || !containsString(t.EnumValues, s) {
			return nil, tagValueErr("标签 " + t.Name + " 应为枚举值之一: " + strings.Join(t.EnumValues, "/"))
		}
		return s, nil

	case model.TagTypeArray:
		arr, ok := v.([]interface{})
		if !ok {
			return nil, tagValueErr("标签 " + t.Name + " 应为数组")
		}
		out := make([]interface{}, 0, len(arr))
		for _, item := range arr {
			sub := model.TagDefinition{Name: t.Name + "[]", Type: t.ElementType}
			nv, err := normalizeTagValue(sub, item)
			if err != nil {
				return nil, err
			}
			out = append(out, nv)
		}
		return out, nil

	case model.TagTypeObject:
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil, tagValueErr("标签 " + t.Name + " 应为对象")
		}
		// 必填子字段
		for _, f := range t.Fields {
			if f.Required {
				if _, ok := m[f.Name]; !ok {
					return nil, tagValueErr("标签 " + t.Name + " 缺少必填子字段: " + f.Name)
				}
			}
		}
		out := make(map[string]interface{}, len(m))
		for k, iv := range m {
			var sub *model.TagDefinition
			for i := range t.Fields {
				if t.Fields[i].Name == k {
					sub = &t.Fields[i]
					break
				}
			}
			if sub == nil {
				return nil, tagValueErr("标签 " + t.Name + " 含未知子字段: " + k)
			}
			nv, err := normalizeTagValue(*sub, iv)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	}
	return nil, tagValueErr("不支持的标签类型: " + string(t.Type))
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
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return i, true
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
	case json.Number:
		if f, err := n.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
