package kiwi

import (
	"context"
	"testing"

	"github.com/kneumoin/nepal/internal/model"
)

func TestKiwi_Search_NoKey(t *testing.T) {
	p := New()
	_, err := p.Search(context.Background(), model.Query{From: "DXB", To: "KTM"})
	if err == nil {
		t.Fatal("expected error without API key")
	}
}

func TestKiwi_Capabilities(t *testing.T) {
	c := New().Capabilities()
	if !c.SupportedAirlines["FZ"] {
		t.Fatal("expected FZ")
	}
}
