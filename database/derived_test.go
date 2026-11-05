package database

import (
	"strings"
	"testing"
	"time"
)

// The nick and the placeholder address are functions of the protocol ids, so they are not stored.
func TestDerivedNickAndEmail(t *testing.T) {
	nick := DerivedNick(12345, "RMCE")
	if !strings.HasSuffix(nick, "RMCE") || len(nick) <= 4 {
		t.Fatalf("nick %q", nick)
	}
	if DerivedNick(12345, "RMCE") != nick || DerivedNick(12346, "RMCE") == nick {
		t.Fatal("nick must be stable and per user")
	}
	if DerivedEmail(12345, "RMCE") != nick+"@nds" {
		t.Fatal("email")
	}
}

func TestIPRetentionIsNinetyDays(t *testing.T) {
	if IPRetention != 90*24*time.Hour {
		t.Fatal(IPRetention)
	}
}
