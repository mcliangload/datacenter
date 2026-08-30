package service

import (
	"os"
	"path/filepath"
	"testing"

	"datacenter/internal/model"
)

func TestMergeTagSchemas(t *testing.T) {
	existing := []model.TagDefinition{
		{Name: "a", Type: model.TagTypeString},
		{Name: "b", Type: model.TagTypeInt},
	}
	incoming := []model.TagDefinition{
		{Name: "b", Type: model.TagTypeFloat}, // 同名替换
		{Name: "c", Type: model.TagTypeBool},  // 新增
	}
	merged := mergeTagSchemas(existing, incoming)
	if len(merged) != 3 {
		t.Fatalf("合并后应为 3 个标签，实际 %d", len(merged))
	}
	byName := make(map[string]model.TagType, len(merged))
	for _, td := range merged {
		byName[td.Name] = td.Type
	}
	if byName["a"] != model.TagTypeString || byName["b"] != model.TagTypeFloat || byName["c"] != model.TagTypeBool {
		t.Fatalf("合并结果不正确: %#v", byName)
	}
}

func TestValidateScriptPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "scrape.sh")
	if err := os.WriteFile(file, []byte("#!/bin/sh\necho {}"), 0o755); err != nil {
		t.Fatalf("创建脚本失败: %v", err)
	}
	subdir := filepath.Join(dir, "d")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"合法脚本文件", file, false},
		{"空路径", "", true},
		{"相对路径", "scripts/a.sh", true},
		{"不存在的绝对路径", filepath.Join(dir, "nope.sh"), true},
		{"目录路径", subdir, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateScriptPath(c.path)
			if c.wantErr && err == nil {
				t.Fatal("期望报错")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("期望成功: %v", err)
			}
		})
	}
}
