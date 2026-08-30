package model

import "testing"

func TestValidDeletePolicy(t *testing.T) {
	valid := []DeletePolicy{
		{Children: PolicyDeny, Incoming: PolicyDeny},
		{Children: PolicyCascade, Incoming: PolicyDetach},
		{Children: PolicyDetach, Incoming: PolicyDeny},
	}
	for _, p := range valid {
		if !ValidDeletePolicy(p) {
			t.Errorf("策略应合法: %+v", p)
		}
	}
	invalid := []DeletePolicy{
		{Children: "xxx", Incoming: PolicyDeny},
		{Children: PolicyDeny, Incoming: "xxx"},
		{Children: "", Incoming: ""},
		{Children: PolicyCascade, Incoming: PolicyCascade}, // incoming 不支持 cascade
	}
	for _, p := range invalid {
		if ValidDeletePolicy(p) {
			t.Errorf("策略应非法: %+v", p)
		}
	}
}

func TestDefaultDeletePolicy(t *testing.T) {
	p := DefaultDeletePolicy()
	if p.Children != PolicyDeny || p.Incoming != PolicyDeny {
		t.Errorf("默认策略应为 deny/deny，实际 %+v", p)
	}
}

func TestEffectiveDeletePolicy(t *testing.T) {
	c := &BusinessCollection{}
	if p := c.EffectiveDeletePolicy(); p != DefaultDeletePolicy() {
		t.Errorf("nil 策略应返回默认，实际 %+v", p)
	}
	dp := DeletePolicy{Children: PolicyCascade, Incoming: PolicyDetach}
	c2 := &BusinessCollection{DeletePolicy: &dp}
	if p := c2.EffectiveDeletePolicy(); p.Children != PolicyCascade || p.Incoming != PolicyDetach {
		t.Errorf("应返回配置策略，实际 %+v", p)
	}
}
