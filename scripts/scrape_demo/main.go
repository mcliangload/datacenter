// scrape_demo 冒烟测试用「刮削脚本」：无论输入什么路径，固定输出一组 JSON 标签。
// 实际生产环境为 NFS 上的 shell/python 脚本，由集合管理员注册（仅路径 + 输出 JSON 的约定）。
package main

import (
	"fmt"
	"os"
)

func main() {
	// 入参：数据路径（Q3：仅一个路径入参）
	path := "<none>"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	fmt.Printf(`{"model_name":"demo-model","version":"1.2","age":3,"stage":"test","config":{"version":"2.0","accuracy":0.98}}`)
	_ = path
}
