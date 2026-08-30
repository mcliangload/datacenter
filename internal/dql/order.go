package dql

import (
	"fmt"
	"regexp"
	"strings"
)

// orderByRe 匹配语句尾部的 ORDER BY 子句（系统优化 1.2）：
//
//	... ORDER BY node DESC
//	... ORDER BY "node" ASC
//
// 字段支持中文/引号包裹；方向缺省为 ASC。
var orderByRe = regexp.MustCompile(`(?is)^(.*?)\s+ORDER\s+BY\s+("(?:[^"\\]|\\.)*"|'[^']*'|[^\s"']+)\s*(ASC|DESC)?\s*$`)

// OrderBy 排序子句
type OrderBy struct {
	Field string
	Desc  bool
}

// ExtractOrderBy 从 DQL 语句尾部提取 ORDER BY 子句，返回剥离后的语句与排序字段。
// 未含 ORDER BY 时返回 (原语句, nil, nil)。
func ExtractOrderBy(dqlStr string) (string, *OrderBy, error) {
	m := orderByRe.FindStringSubmatch(strings.TrimSpace(dqlStr))
	if m == nil {
		return dqlStr, nil, nil
	}
	field := strings.Trim(m[2], `"'`)
	if field == "" {
		return "", nil, fmt.Errorf("ORDER BY 字段不能为空")
	}
	dir := strings.ToUpper(strings.TrimSpace(m[3]))
	return strings.TrimSpace(m[1]), &OrderBy{Field: field, Desc: dir == "DESC"}, nil
}
