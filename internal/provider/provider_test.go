package provider_test

import (
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/provider"
)

func TestProviderNew(t *testing.T) {
	p := provider.New("test")
	if p == nil {
		t.Fatal("expected non-nil provider factory")
	}
	instance := p()
	if instance == nil {
		t.Fatal("expected non-nil provider instance")
	}
}
