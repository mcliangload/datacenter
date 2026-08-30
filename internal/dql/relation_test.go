package dql

import (
	"testing"
)

func TestExtractRelationRefs(t *testing.T) {
	// parent 直接子
	n, err := Parse(`parent = "6a84710e19ef6e71184fd558" AND age = 1`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	refs, err := ExtractRelationRefs(n)
	if err != nil {
		t.Fatalf("提取失败: %v", err)
	}
	if len(refs) != 1 || refs[0].Field != "parent" || refs[0].Op != "=" || refs[0].Values[0] != "6a84710e19ef6e71184fd558" {
		t.Fatalf("提取不正确: %#v", refs)
	}

	// ancestor 子树（IN）
	n2, err := Parse(`ancestor IN ("a", "b")`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	refs2, err := ExtractRelationRefs(n2)
	if err != nil || len(refs2) != 1 || refs2[0].Field != "ancestor" || len(refs2[0].Values) != 2 {
		t.Fatalf("IN 提取不正确: %#v err=%v", refs2, err)
	}

	// 非法运算符
	n3, err := Parse(`parent > "x"`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if _, err := ExtractRelationRefs(n3); err == nil {
		t.Fatal("parent 范围运算符应被拒绝")
	}
}

func TestBuildFilter_SkipRelationFields(t *testing.T) {
	schemas := testSchemas()
	// parent/ancestor 条件不进入标签过滤（由服务层解析）
	for _, dqlStr := range []string{`parent = "abc" AND age = 1`, `ancestor = "abc"`} {
		node, err := Parse(dqlStr)
		if err != nil {
			t.Fatalf("解析失败 %q: %v", dqlStr, err)
		}
		if _, err := BuildFilter(node, schemas); err != nil {
			t.Fatalf("构建失败 %q: %v", dqlStr, err)
		}
	}
}
