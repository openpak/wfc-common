package database

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewPairingCodeShape(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		c := NewPairingCode()
		if got, ok := NormalizePairingCode(c); !ok || got != c {
			t.Fatalf("generated code %q does not round-trip", c)
		}
		seen[c] = true
	}
	if len(seen) < 495 {
		t.Fatalf("codes collide too much: %d distinct of 500", len(seen))
	}
}

func TestNormalizePairingCode(t *testing.T) {
	for _, in := range []string{"abcd-2345", " ABCD 2345 ", "abcd2345"} {
		if c, ok := NormalizePairingCode(in); !ok || c != "ABCD2345" {
			t.Errorf("%q -> %q %v", in, c, ok)
		}
	}
	for _, in := range []string{"", "ABCD234", "ABCD23456", "ABCD-23I5", "ABCD-2305", "ABC$2345"} {
		if _, ok := NormalizePairingCode(in); ok {
			t.Errorf("%q should be rejected", in)
		}
	}
	if FormatPairingCode("ABCD2345") != "ABCD-2345" {
		t.Error("format")
	}
}

// TestPairingLifecycle runs against a real Postgres when WFC_TEST_DSN is set.
func TestPairingLifecycle(t *testing.T) {
	dsn := os.Getenv("WFC_TEST_DSN")
	if dsn == "" {
		t.Skip("WFC_TEST_DSN not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	c := &Connection{pool: pool, ctx: ctx}
	if err := c.EnsurePairing(); err != nil {
		t.Fatal(err)
	}
	if err := c.EnsurePairing(); err != nil { // idempotent
		t.Fatal(err)
	}
	const uid = 424242
	_, _ = pool.Exec(ctx, `DELETE FROM wfc_pairing_codes WHERE user_id=$1`, uid)
	_ = c.ForgetPairing(ctx, uid)

	code, err := c.IssuePairingCode(ctx, uid)
	if err != nil || code == "" {
		t.Fatalf("issue: %q %v", code, err)
	}
	again, _ := c.IssuePairingCode(ctx, uid)
	if again != code {
		t.Fatalf("a live code must be reused: %q vs %q", again, code)
	}
	if got, err := c.LookupPairingCode(ctx, strings.ToLower(FormatPairingCode(code))); err != nil || got.UserID != uid {
		t.Fatalf("lookup: %+v %v", got, err)
	}
	if _, err := c.LookupPairingCode(ctx, "AAAAAAAA"); err != ErrPairingCode {
		t.Fatalf("unknown: %v", err)
	}
	if got, err := c.RedeemPairingCode(ctx, code); err != nil || got != uid {
		t.Fatalf("redeem: %d %v", got, err)
	}
	if _, err := c.RedeemPairingCode(ctx, code); err != ErrPairingCode {
		t.Fatalf("single use: %v", err)
	}
	if got, err := c.IssuePairingCode(ctx, uid); err != nil || got != "" {
		t.Fatalf("linked console must get no code: %q %v", got, err)
	}
	if err := c.ForgetPairing(ctx, uid); err != nil {
		t.Fatal(err)
	}
	next, err := c.IssuePairingCode(ctx, uid)
	if err != nil || next == "" || next == code {
		t.Fatalf("after forget: %q %v", next, err)
	}
	// Expiry.
	_, _ = pool.Exec(ctx, `UPDATE wfc_pairing_codes SET expires_at = now() - interval '1 minute' WHERE code=$1`, next)
	if _, err := c.RedeemPairingCode(ctx, next); err != ErrPairingCode {
		t.Fatalf("expired: %v", err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM wfc_pairing_codes WHERE user_id=$1`, uid)
	_ = c.ForgetPairing(ctx, uid)
}
