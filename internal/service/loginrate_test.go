package service

import "testing"

// 登录防爆破：阈值 5 次失败锁定，成功清零
func TestLoginGuard(t *testing.T) {
	g := newLoginGuard()
	key := "127.0.0.1|admin"

	// 前 4 次失败仍允许
	for i := 0; i < loginMaxFails-1; i++ {
		if !g.allow(key) {
			t.Fatalf("第 %d 次失败后不应锁定（阈值 %d）", i+1, loginMaxFails)
		}
		g.fail(key)
	}
	// 第 5 次失败后锁定
	if !g.allow(key) {
		t.Fatalf("第 %d 次尝试前不应锁定", loginMaxFails)
	}
	g.fail(key)
	if g.allow(key) {
		t.Fatal("达到阈值后应锁定")
	}
	if g.allow(key) {
		t.Fatal("锁定窗口内应持续拒绝")
	}

	// 成功登录清零
	g.success(key)
	if !g.allow(key) {
		t.Fatal("成功登录后应解除锁定")
	}

	// 不同 key 互不影响
	if !g.allow("127.0.0.2|admin") {
		t.Fatal("不同 IP 不应互相影响")
	}
}
