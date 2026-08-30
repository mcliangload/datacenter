package service

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"datacenter/internal/model"
)

func TestValidateTagSchema_Valid(t *testing.T) {
	schema := []model.TagDefinition{
		{Name: "name", Type: model.TagTypeString, Required: true},
		{Name: "age", Type: model.TagTypeInt},
		{Name: "stage", Type: model.TagTypeEnum, EnumValues: []string{"a", "b"}},
		{Name: "list", Type: model.TagTypeArray, ElementType: model.TagTypeString},
		{Name: "cfg", Type: model.TagTypeObject, Fields: []model.TagDefinition{
			{Name: "v", Type: model.TagTypeFloat, Required: true},
			{Name: "sub", Type: model.TagTypeObject, Fields: []model.TagDefinition{{Name: "x", Type: model.TagTypeInt}}},
		}},
		{Name: "ok", Type: model.TagTypeBool},
		{Name: "day", Type: model.TagTypeDate},
	}
	if err := ValidateTagSchema(schema); err != nil {
		t.Fatalf("期望通过校验，实际: %v", err)
	}
}

func TestValidateTagSchema_Invalid(t *testing.T) {
	cases := []struct {
		name   string
		schema []model.TagDefinition
	}{
		{"空标签名", []model.TagDefinition{{Name: "  ", Type: model.TagTypeString}}},
		{"保留字段冲突", []model.TagDefinition{{Name: "path", Type: model.TagTypeString}}},
		{"重复标签名", []model.TagDefinition{{Name: "a", Type: model.TagTypeString}, {Name: "a", Type: model.TagTypeInt}}},
		{"不支持的标签类型", []model.TagDefinition{{Name: "a", Type: "unknown"}}},
		{"enum 缺少枚举值", []model.TagDefinition{{Name: "a", Type: model.TagTypeEnum}}},
		{"array 元素类型为对象", []model.TagDefinition{{Name: "a", Type: model.TagTypeArray, ElementType: model.TagTypeObject}}},
		{"object 缺少子字段", []model.TagDefinition{{Name: "a", Type: model.TagTypeObject}}},
		{"object 子字段重复", []model.TagDefinition{{Name: "a", Type: model.TagTypeObject, Fields: []model.TagDefinition{
			{Name: "x", Type: model.TagTypeString}, {Name: "x", Type: model.TagTypeInt},
		}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := ValidateTagSchema(c.schema); err == nil {
				t.Fatal("期望校验失败")
			}
		})
	}
}

func testSchema() []model.TagDefinition {
	return []model.TagDefinition{
		{Name: "name", Type: model.TagTypeString, Required: true},
		{Name: "age", Type: model.TagTypeInt, Required: true},
		{Name: "score", Type: model.TagTypeFloat},
		{Name: "ok", Type: model.TagTypeBool},
		{Name: "day", Type: model.TagTypeDate},
		{Name: "stage", Type: model.TagTypeEnum, EnumValues: []string{"a", "b"}},
		{Name: "list", Type: model.TagTypeArray, ElementType: model.TagTypeInt},
		{Name: "cfg", Type: model.TagTypeObject, Fields: []model.TagDefinition{
			{Name: "v", Type: model.TagTypeString, Required: true},
			{Name: "n", Type: model.TagTypeInt},
		}},
	}
}

func TestValidateAndNormalizeTags(t *testing.T) {
	schema := testSchema()

	t.Run("缺必填标签", func(t *testing.T) {
		_, err := ValidateAndNormalizeTags(schema, map[string]interface{}{"name": "x"}, false)
		if err == nil {
			t.Fatal("期望失败：age 为必填")
		}
	})

	t.Run("未知标签拒绝（手动场景）", func(t *testing.T) {
		_, err := ValidateAndNormalizeTags(schema, map[string]interface{}{"name": "x", "age": 1, "zzz": 2}, false)
		if err == nil {
			t.Fatal("期望失败：未知标签")
		}
	})

	t.Run("未知标签忽略（刮削场景）", func(t *testing.T) {
		out, err := ValidateAndNormalizeTags(schema, map[string]interface{}{"name": "x", "age": 1, "zzz": 2}, true)
		if err != nil {
			t.Fatalf("期望成功: %v", err)
		}
		if _, ok := out["zzz"]; ok {
			t.Fatal("未知标签应被忽略")
		}
	})

	t.Run("类型规范化", func(t *testing.T) {
		out, err := ValidateAndNormalizeTags(schema, map[string]interface{}{
			"name": "x", "age": 3.0, "score": 1, "ok": true,
			"day": "2024-01-02", "stage": "a", "list": []interface{}{1, 2.0, 3},
			"cfg": map[string]interface{}{"v": "1.0", "n": 2},
		}, false)
		if err != nil {
			t.Fatalf("期望成功: %v", err)
		}
		if age, ok := out["age"].(int64); !ok || age != 3 {
			t.Errorf("age 应规范化为 int64(3)，实际 %#v", out["age"])
		}
		if score, ok := out["score"].(float64); !ok || score != 1 {
			t.Errorf("score 应规范化为 float64(1)，实际 %#v", out["score"])
		}
		if _, ok := out["day"].(time.Time); !ok {
			t.Errorf("day 应规范化为 time.Time，实际 %#v", out["day"])
		}
		list, ok := out["list"].([]interface{})
		if !ok || len(list) != 3 {
			t.Fatalf("list 应为 3 元素，实际 %#v", out["list"])
		}
		if list[1].(int64) != 2 {
			t.Errorf("list[1] 应规范化为 int64(2)，实际 %#v", list[1])
		}
	})

	t.Run("enum 非法值", func(t *testing.T) {
		_, err := ValidateAndNormalizeTags(schema, map[string]interface{}{"name": "x", "age": 1, "stage": "zzz"}, false)
		if err == nil {
			t.Fatal("期望失败：enum 值不合法")
		}
	})

	t.Run("array 元素类型校验", func(t *testing.T) {
		_, err := ValidateAndNormalizeTags(schema, map[string]interface{}{"name": "x", "age": 1, "list": []interface{}{1, "s"}}, false)
		if err == nil {
			t.Fatal("期望失败：数组元素类型不符")
		}
	})

	t.Run("object 缺必填子字段", func(t *testing.T) {
		_, err := ValidateAndNormalizeTags(schema, map[string]interface{}{"name": "x", "age": 1, "cfg": map[string]interface{}{}}, false)
		if err == nil {
			t.Fatal("期望失败：缺必填子字段 v")
		}
	})

	t.Run("object 未知子字段", func(t *testing.T) {
		_, err := ValidateAndNormalizeTags(schema, map[string]interface{}{"name": "x", "age": 1, "cfg": map[string]interface{}{"v": "1", "zz": 1}}, false)
		if err == nil {
			t.Fatal("期望失败：未知子字段")
		}
	})

	t.Run("json.Number 兼容（刮削输出 UseNumber）", func(t *testing.T) {
		raw := []byte(`{"name":"x","age":3,"day":"2024-01-02T00:00:00Z","stage":"a"}`)
		var m map[string]interface{}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&m); err != nil {
			t.Fatalf("JSON 解析失败: %v", err)
		}
		out, err := ValidateAndNormalizeTags(schema, m, false)
		if err != nil {
			t.Fatalf("期望成功: %v", err)
		}
		if out["age"].(int64) != 3 {
			t.Errorf("json.Number 应规范化为 int64(3)，实际 %#v", out["age"])
		}
	})
}
