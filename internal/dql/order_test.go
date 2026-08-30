package dql

import "testing"

func TestExtractOrderBy(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantField string
		wantDesc  bool
		wantHas   bool
	}{
		{"无排序子句", `collection = "model"`, "", false, false},
		{"ASC 缺省方向", `node = "7" ORDER BY accuracy_rms`, "accuracy_rms", false, true},
		{"DESC 方向", `node = "7" ORDER BY accuracy_rms DESC`, "accuracy_rms", true, true},
		{"引号字段", `node = "7" ORDER BY "accuracy_rms" ASC`, "accuracy_rms", false, true},
		{"中文字段", `collection = "模型" ORDER BY 精度 desc`, "精度", true, true},
		{"带分页无排序", `node = "7"`, "", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clean, ob, err := ExtractOrderBy(c.in)
			if err != nil {
				t.Fatalf("ExtractOrderBy 错误: %v", err)
			}
			if (ob != nil) != c.wantHas {
				t.Fatalf("存在排序=%v, 期望 %v（clean=%q）", ob != nil, c.wantHas, clean)
			}
			if ob != nil {
				if ob.Field != c.wantField || ob.Desc != c.wantDesc {
					t.Fatalf("排序=%+v, 期望 field=%s desc=%v", ob, c.wantField, c.wantDesc)
				}
				if clean == c.in {
					t.Fatal("ORDER BY 未从语句中剥离")
				}
			}
		})
	}
}
