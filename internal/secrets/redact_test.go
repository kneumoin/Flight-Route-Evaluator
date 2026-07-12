package secrets_test

import (
	"strings"
	"testing"

	"github.com/kneumoin/nepal/internal/secrets"
)

func TestRedactTokenInURL(t *testing.T) {
	token := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	s := secrets.Redact("https://api.example.com?token="+token+"&origin=MOW", token)
	if strings.Contains(s, token) {
		t.Fatal("token leaked in redacted string")
	}
}

func TestTravelpayoutsTokenPrefersEnv(t *testing.T) {
	t.Setenv("TRAVELPAYOUTS_TOKEN", "tp-token")
	t.Setenv("AVIASALES_TOKEN", "av-token")
	if got := secrets.TravelpayoutsToken(""); got != "tp-token" {
		t.Fatalf("expected TRAVELPAYOUTS_TOKEN, got %q", got)
	}
}

func TestTravelpayoutsTokenOverride(t *testing.T) {
	t.Setenv("TRAVELPAYOUTS_TOKEN", "tp-token")
	if got := secrets.TravelpayoutsToken("cli-token"); got != "cli-token" {
		t.Fatalf("expected CLI override, got %q", got)
	}
}
