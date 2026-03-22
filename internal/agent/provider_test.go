package agent

import "testing"

func TestGetProvider_Zhipu(t *testing.T) {
	p, ok := GetProvider("zhipu")
	if !ok {
		t.Fatal("expected zhipu provider to exist")
	}
	if p.Model != "GLM-5" {
		t.Fatalf("expected model GLM-5, got %s", p.Model)
	}
	if p.BaseURL != "https://open.bigmodel.cn/api/anthropic" {
		t.Fatalf("unexpected base URL: %s", p.BaseURL)
	}
}

func TestGetProvider_MiniMax(t *testing.T) {
	p, ok := GetProvider("minimax")
	if !ok {
		t.Fatal("expected minimax provider to exist")
	}
	if p.Model != "MiniMax-M2.7" {
		t.Fatalf("expected model MiniMax-M2.7, got %s", p.Model)
	}
}

func TestGetProvider_Unknown(t *testing.T) {
	_, ok := GetProvider("nonexistent")
	if ok {
		t.Fatal("expected unknown provider to return false")
	}
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	if len(providers) < 2 {
		t.Fatalf("expected at least 2 providers, got %d", len(providers))
	}
}
