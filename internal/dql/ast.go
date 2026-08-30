package dql

// Node DQL 抽象语法树节点
type Node interface{ isNode() }

// Cond 条件：字段 运算符 值
type Cond struct {
	Field string      // 字段名（标签名或 collection 特殊字段）
	Op    string      // = != > >= < <= IN EXISTS LIKE
	Value interface{} // 标量 / []interface{}（IN）/ bool（EXISTS）
}

// And 逻辑与（AND 优先级高于 OR）
type And struct{ Left, Right Node }

// Or 逻辑或
type Or struct{ Left, Right Node }

func (*Cond) isNode() {}
func (*And) isNode()  {}
func (*Or) isNode()   {}
