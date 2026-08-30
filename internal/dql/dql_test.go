package dql

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"datacenter/internal/model"
)

func TestParse_PrecedenceAndParens(t *testing.T) {
	// AND 优先于 OR：a = 1 AND b = 2 OR c = 3 → Or{And{a,b}, c}
	n, err := Parse(`a = 1 AND b = 2 OR c = 3`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	or, ok := n.(*Or)
	if !ok {
		t.Fatalf("期望 Or 根节点，实际 %T", n)
	}
	if _, ok := or.Left.(*And); !ok {
		t.Fatalf("期望 Or.Left 为 And，实际 %T", or.Left)
	}

	// 括号改变优先级：(a = 1 OR b = 2) AND c = 3 → And{Or{a,b}, c}
	n2, err := Parse(`(a = 1 OR b = 2) AND c = 3`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	and, ok := n2.(*And)
	if !ok {
		t.Fatalf("期望 And 根节点，实际 %T", n2)
	}
	if _, ok := and.Left.(*Or); !ok {
		t.Fatalf("期望 And.Left 为 Or，实际 %T", and.Left)
	}
}

func TestParse_Literals(t *testing.T) {
	cases := []struct {
		name string
		dql  string
		want interface{}
	}{
		{"单引号字符串", `a = 'x y'`, "x y"},
		{"双引号字符串", `a = "x"`, "x"},
		{"裸字符串值", `a = demo`, "demo"},
		{"整数", `a = 42`, int64(42)},
		{"负数", `a = -3`, int64(-3)},
		{"浮点", `a = 1.5`, 1.5},
		{"布尔", `a = true`, true},
		{"引号转义", `a = "he\"llo"`, `he"llo`},
		{"中文标签名", `模型名 = "demo"`, "demo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, err := Parse(c.dql)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			cond, ok := n.(*Cond)
			if !ok {
				t.Fatalf("期望 Cond，实际 %T", n)
			}
			if !reflect.DeepEqual(cond.Value, c.want) {
				t.Fatalf("值不匹配: 期望 %#v 实际 %#v", c.want, cond.Value)
			}
		})
	}
}

func TestParse_INAndExists(t *testing.T) {
	n, err := Parse(`a IN ("x", 'y', 3)`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	cond := n.(*Cond)
	vals, ok := cond.Value.([]interface{})
	if !ok || len(vals) != 3 {
		t.Fatalf("IN 值列表不正确: %#v", cond.Value)
	}

	n2, err := Parse(`b EXISTS true`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if v := n2.(*Cond).Value; v != true {
		t.Fatalf("EXISTS 值应为 true，实际 %#v", v)
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []string{
		``,
		`a =`,
		`a = "未闭合`,
		`(a = 1`,
		`a = 1 b = 2`,      // 缺少 AND/OR
		`a = 1 AND`,        // 尾部缺条件
		`AND a = 1`,        // 开头缺条件
		`a ~ 1`,            // 非法运算符
		`!a = 1`,           // 非法字符
		`a = 1 AND b = 2)`, // 多余右括号
		`a IN ()`,          // 空 IN 列表
		`a EXISTS "x"`,     // EXISTS 非布尔
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, err := Parse(c); err == nil {
				t.Fatalf("语句 %q 应解析失败", c)
			}
		})
	}
}

func testSchemas() map[string][]model.TagDefinition {
	return map[string][]model.TagDefinition{
		"c1": {
			{Name: "name", Type: model.TagTypeString},
			{Name: "age", Type: model.TagTypeInt},
			{Name: "score", Type: model.TagTypeFloat},
			{Name: "ok", Type: model.TagTypeBool},
			{Name: "day", Type: model.TagTypeDate},
			{Name: "stage", Type: model.TagTypeEnum, EnumValues: []string{"dev", "test"}},
		},
		"c2": {
			{Name: "name", Type: model.TagTypeString},
			{Name: "age", Type: model.TagTypeInt},
		},
	}
}

func TestBuildFilter(t *testing.T) {
	schemas := testSchemas()

	cases := []struct {
		name string
		dql  string
		want bson.M
	}{
		{"等值 int", `age = 3`, bson.M{"tags.age": int64(3)}},
		{"等值 string", `name = "demo"`, bson.M{"tags.name": "demo"}},
		{"等值 float", `score = 1.5`, bson.M{"tags.score": 1.5}},
		{"ne", `age != 3`, bson.M{"tags.age": bson.M{"$ne": int64(3)}}},
		{"范围 gte", `age >= 3`, bson.M{"tags.age": bson.M{"$gte": int64(3)}}},
		{"范围 lt float", `score < 2`, bson.M{"tags.score": bson.M{"$lt": 2.0}}},
		{"IN 混合", `stage IN ("dev", "prod")`, bson.M{"tags.stage": bson.M{"$in": []interface{}{"dev", "prod"}}}},
		{"EXISTS true", `name EXISTS true`, bson.M{"tags.name": bson.M{"$exists": true}}},
		{"LIKE 转义", `name LIKE "a.b"`, bson.M{"tags.name": bson.M{"$regex": "a\\.b", "$options": "i"}}},
		{"日期", `day = "2024-01-02"`, bson.M{"tags.day": time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)}},
		{"AND", `name = "a" AND age = 1`, bson.M{"$and": []bson.M{{"tags.name": "a"}, {"tags.age": int64(1)}}}},
		{"OR", `name = "a" OR age = 1`, bson.M{"$or": []bson.M{{"tags.name": "a"}, {"tags.age": int64(1)}}}},
		{"括号", `(name = "a" OR age = 1) AND ok = true`,
			bson.M{"$and": []bson.M{
				bson.M{"$or": []bson.M{{"tags.name": "a"}, {"tags.age": int64(1)}}},
				{"tags.ok": true},
			}}},
		{"collection 条件被跳过", `collection = "x" AND age = 1`, bson.M{"tags.age": int64(1)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			node, err := Parse(c.dql)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			got, err := BuildFilter(node, schemas)
			if err != nil {
				t.Fatalf("构建失败: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("过滤条件不匹配\n期望: %#v\n实际: %#v", c.want, got)
			}
		})
	}
}

func TestBuildFilter_Errors(t *testing.T) {
	schemas := testSchemas()
	cases := []struct {
		name string
		dql  string
	}{
		{"未知标签", `zzz = 1`},
		{"int 类型不匹配", `age = "x"`},
		{"string 范围查询", `name > "a"`},
		{"LIKE 非字符串字段", `age LIKE "1"`},
		{"bool 类型不匹配", `ok = 1`},
		{"日期格式错误", `day = "not-a-date"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			node, err := Parse(c.dql)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			if _, err := BuildFilter(node, schemas); err == nil {
				t.Fatalf("语句 %q 应构建失败", c.dql)
			}
		})
	}
}

func TestExtractCollections(t *testing.T) {
	n, err := Parse(`collection = "模型库" AND age = 1`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	refs, err := ExtractCollections(n)
	if err != nil {
		t.Fatalf("提取失败: %v", err)
	}
	if len(refs) != 1 || refs[0].Op != "=" || refs[0].Names[0] != "模型库" {
		t.Fatalf("提取不正确: %#v", refs)
	}

	// 非法运算符
	n2, err := Parse(`collection > "x"`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if _, err := ExtractCollections(n2); err == nil {
		t.Fatal("collection 范围运算符应被拒绝")
	}

	// IN
	n3, err := Parse(`collection IN ("a", "b")`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	refs3, err := ExtractCollections(n3)
	if err != nil || len(refs3) != 1 || refs3[0].Op != "IN" || len(refs3[0].Names) != 2 {
		t.Fatalf("IN 提取不正确: %#v err=%v", refs3, err)
	}
}
