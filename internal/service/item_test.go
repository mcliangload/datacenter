package service

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"datacenter/internal/model"
)

func TestBuildTagFilter(t *testing.T) {
	s := &ItemService{}
	schema := []model.TagDefinition{
		{Name: "name", Type: model.TagTypeString},
		{Name: "age", Type: model.TagTypeInt},
		{Name: "score", Type: model.TagTypeFloat},
		{Name: "ok", Type: model.TagTypeBool},
		{Name: "day", Type: model.TagTypeDate},
		{Name: "stage", Type: model.TagTypeEnum, EnumValues: []string{"a", "b"}},
		{Name: "list", Type: model.TagTypeArray, ElementType: model.TagTypeString},
	}

	cases := []struct {
		name    string
		params  url.Values
		want    bson.M
		wantErr bool
	}{
		{"等值 string", url.Values{"name": {"x"}, "page": {"1"}, "page_size": {"20"}}, bson.M{"tags.name": "x"}, false},
		{"等值 int", url.Values{"age": {"3"}}, bson.M{"tags.age": int64(3)}, false},
		{"等值 float", url.Values{"score": {"1.5"}}, bson.M{"tags.score": 1.5}, false},
		{"等值 bool", url.Values{"ok": {"true"}}, bson.M{"tags.ok": true}, false},
		{"等值 date", url.Values{"day": {"2024-01-02"}}, bson.M{"tags.day": time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)}, false},
		{"范围 gte", url.Values{"age.gte": {"3"}}, bson.M{"tags.age": bson.M{"$gte": int64(3)}}, false},
		{"范围 lt", url.Values{"score.lt": {"1.5"}}, bson.M{"tags.score": bson.M{"$lt": 1.5}}, false},
		{"ne", url.Values{"age.ne": {"5"}}, bson.M{"tags.age": bson.M{"$ne": int64(5)}}, false},
		{"in", url.Values{"stage.in": {"a,b"}}, bson.M{"tags.stage": bson.M{"$in": []interface{}{"a", "b"}}}, false},
		{"exists", url.Values{"name.exists": {"true"}}, bson.M{"tags.name": bson.M{"$exists": true}}, false},
		{"多条件组合", url.Values{"name": {"x"}, "age.gte": {"1"}}, bson.M{"tags.name": "x", "tags.age": bson.M{"$gte": int64(1)}}, false},
		{"未知标签", url.Values{"zzz": {"1"}}, nil, true},
		{"非法操作符", url.Values{"age.regex": {"1"}}, nil, true},
		{"int 解析失败", url.Values{"age": {"abc"}}, nil, true},
		{"array 标签不支持查询", url.Values{"list": {"a"}}, nil, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := s.buildTagFilter(schema, c.params)
			if c.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际 filter=%v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("filter 不匹配\n期望: %#v\n实际: %#v", c.want, got)
			}
		})
	}
}

func TestParseQueryValue(t *testing.T) {
	if v, err := parseQueryValue(model.TagDefinition{Name: "n", Type: model.TagTypeString}, "x"); err != nil || v != "x" {
		t.Errorf("string 解析失败: %v %v", v, err)
	}
	if v, err := parseQueryValue(model.TagDefinition{Name: "n", Type: model.TagTypeInt}, "42"); err != nil || v != int64(42) {
		t.Errorf("int 解析失败: %v %v", v, err)
	}
	if _, err := parseQueryValue(model.TagDefinition{Name: "n", Type: model.TagTypeInt}, "4.2"); err == nil {
		t.Error("int 应拒绝非整数")
	}
	if _, err := parseQueryValue(model.TagDefinition{Name: "n", Type: model.TagTypeArray, ElementType: model.TagTypeString}, "a"); err == nil {
		t.Error("array 类型应拒绝等值查询")
	}
	if _, err := parseQueryValue(model.TagDefinition{Name: "n", Type: "unknown"}, "a"); err == nil {
		t.Error("未知类型应报错")
	}
}

func TestValidatePath(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	file := filepath.Join(root, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.txt")

	cases := []struct {
		name    string
		svc     *ItemService
		path    string
		wantErr bool
	}{
		{"根目录内文件", &ItemService{dataRoot: root}, file, false},
		{"根目录内目录", &ItemService{dataRoot: root}, sub, false},
		{"根目录外（即使存在）", &ItemService{dataRoot: root}, outside, true},
		{"根目录内不存在", &ItemService{dataRoot: root}, filepath.Join(root, "nope"), true},
		{"相对路径", &ItemService{dataRoot: root}, "rel/path", true},
		{"空路径", &ItemService{dataRoot: root}, "", true},
		{"未配置根目录", &ItemService{}, file, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.svc.validatePath(c.path)
			if c.wantErr && err == nil {
				t.Fatal("期望报错")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("期望成功: %v", err)
			}
		})
	}
}
