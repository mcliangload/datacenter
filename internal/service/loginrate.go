package service

import (
	"sync"
	"time"
)

// 登录防爆破参数（安全增强 P0-1）：
// 同一（客户端IP, 用户名）连续失败 loginMaxFails 次后，锁定 loginLockWindow 时长。
const (
	loginMaxFails   = 5
	loginLockWindow = 15 * time.Minute
)

// loginGuard 内存级登录失败计数（单实例部署；多实例需分布式计数，见 安全增强方案.md §4.1）。
// 进程重启后计数清零（可接受）；成功登录清零；过期记录惰性清理。
type loginGuard struct {
	mu    sync.Mutex
	fails map[string][]time.Time // key = 客户端IP|用户名 → 失败时间戳列表
}

func newLoginGuard() *loginGuard {
	return &loginGuard{fails: map[string][]time.Time{}}
}

// allow 返回是否允许该 key 尝试登录（false = 处于锁定窗口内）
func (g *loginGuard) allow(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	recs := g.fails[key]
	valid := recs[:0]
	for _, t := range recs {
		if now.Sub(t) < loginLockWindow {
			valid = append(valid, t)
		}
	}
	g.fails[key] = valid
	return len(valid) < loginMaxFails
}

// fail 记录一次失败
func (g *loginGuard) fail(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fails[key] = append(g.fails[key], time.Now())
}

// success 登录成功清零
func (g *loginGuard) success(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.fails, key)
}
