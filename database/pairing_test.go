package database

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestParsePairingInput(t *testing.T) {
	if c, n, ok := ParsePairingInput("abcd-2345"); !ok || c != "ABCD2345" || n != 0 {
		t.Fatalf("code: %q %d %v", c, n, ok)
	}
	if c, n, ok := ParsePairingInput(" 24 829 "); !ok || c != "" || n != 4829 {
		t.Fatalf("number: %q %d %v", c, n, ok)
	}
	for _, in := range []string{"23999", "30000", "4829", "482913", "ABCD", ""} {
		if _, _, ok := ParsePairingInput(in); ok {
			t.Errorf("%q should be rejected", in)
		}
	}
	for i := 0; i < 2000; i++ {
		if n := NewLinkNumber(); n < linkNumberMin || n > linkNumberMax {
			t.Fatalf("number out of range: %d", n)
		}
	}
	k := []byte("k")
	now := time.Now()
	if !bytes.Equal(NetTag(k, "198.51.100.7:5", now), NetTag(k, "198.51.100.7", now)) {
		t.Fatal("port must not matter")
	}
	if bytes.Equal(NetTag(k, "198.51.100.7", now), NetTag(k, "198.51.100.8", now)) {
		t.Fatal("IPv4 tags are per address")
	}
	if !bytes.Equal(NetTag(k, "2001:db8:1:2::5", now), NetTag(k, "[2001:db8:1:2:aa::9]:443", now)) {
		t.Fatal("IPv6 tags are per /64")
	}
	if NetTag(nil, "198.51.100.7", now) != nil || NetTag(k, "junk", now) != nil {
		t.Fatal("no key or no address: no tag")
	}
	if id, ok := SubjectUserID("123456789.1730000000000"); !ok || id != 123456789 {
		t.Fatal("subject parse")
	}
	if id, ok := SubjectUserID("987"); !ok || id != 987 {
		t.Fatal("bare subject parse")
	}
	if _, ok := SubjectUserID("x.1"); ok {
		t.Fatal("bad subject accepted")
	}
}

func testConn(t *testing.T) (*Connection, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("WFC_TEST_DSN")
	if dsn == "" {
		t.Skip("WFC_TEST_DSN not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	c := &Connection{pool: pool, ctx: ctx}
	if err := c.EnsurePairing(); err != nil {
		t.Fatal(err)
	}
	if err := c.EnsurePairing(); err != nil { // idempotent
		t.Fatal(err)
	}
	return c, pool
}

// TestPairingLifecycle runs against a real Postgres when WFC_TEST_DSN is set.
func TestPairingLifecycle(t *testing.T) {
	c, pool := testConn(t)
	ctx := context.Background()
	const uid = 424242
	clean := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM wfc_pairing_codes WHERE user_id=$1`, uid)
		_ = c.ForgetPairing(ctx, uid)
	}
	clean()
	defer clean()
	key := []byte("test-key")
	home := NetTag(key, "198.51.100.7:1234", time.Now())
	req := PairingRequest{UserID: uid, Unit: UnitWii, GameCode: "RMCJ", Name: "Tobagin", NetTag: home}
	atHome := NumberProof{NetTags: NetTags(key, "198.51.100.7", time.Now())}
	elsewhere := NumberProof{NetTags: NetTags(key, "203.0.113.5", time.Now())}

	p, err := c.IssuePairingCode(ctx, req)
	if err != nil || p.Code == "" || p.Number < linkNumberMin {
		t.Fatalf("issue: %+v %v", p, err)
	}
	again, _ := c.IssuePairingCode(ctx, req)
	if again.Code != p.Code || again.Number != p.Number {
		t.Fatalf("a live code must be reused: %+v vs %+v", again, p)
	}
	num := strconv.Itoa(DisplayNumber(p.Number))
	if _, err := c.LookupPairingCode(ctx, num, elsewhere); err != ErrPairingCode {
		t.Fatalf("a number from another network must not resolve: %v", err)
	}
	if got, err := c.LookupPairingCode(ctx, num, NumberProof{Name: " tobagin "}); err != nil || !got.ByNumber {
		t.Fatalf("a number with the in-game name: %+v %v", got, err)
	}
	for _, in := range []string{strings.ToLower(FormatPairingCode(p.Code)), num} {
		got, err := c.LookupPairingCode(ctx, in, atHome)
		if err != nil || got.UserID != uid || got.Unit != UnitWii || got.GameCode != "RMCJ" || got.Name != "Tobagin" || got.Code != p.Code {
			t.Fatalf("lookup %q: %+v %v", in, got, err)
		}
	}
	if _, err := c.LookupPairingCode(ctx, "AAAAAAAA", NumberProof{}); err != ErrPairingCode {
		t.Fatalf("unknown: %v", err)
	}

	// Approval: another account cannot claim a code mid-approval; the same account resumes
	// with the same subject.
	if _, _, err := c.BeginApproval(ctx, num, "acct-a", elsewhere); err != ErrPairingCode {
		t.Fatalf("approve by number without proof: %v", err)
	}
	t1, linked, err := c.BeginApproval(ctx, num, "acct-a", atHome)
	if err != nil || linked || t1.Subject == "" {
		t.Fatalf("begin: %+v %v %v", t1, linked, err)
	}
	if id, ok := SubjectUserID(t1.Subject); !ok || id != uid {
		t.Fatalf("subject %q", t1.Subject)
	}
	if _, _, err := c.BeginApproval(ctx, p.Code, "acct-b", NumberProof{}); err != ErrPairingCode {
		t.Fatalf("other account mid-approval: %v", err)
	}
	if _, err := c.LookupPairingCode(ctx, p.Code, NumberProof{}); err != ErrPairingCode {
		t.Fatalf("lookup mid-approval: %v", err)
	}
	t2, _, err := c.BeginApproval(ctx, p.Code, "acct-a", NumberProof{})
	if err != nil || t2.Subject != t1.Subject {
		t.Fatalf("resume: %+v %v", t2, err)
	}
	if pend, ok, err := c.PendingApproval(ctx, uid); err != nil || !ok || pend.Subject != t1.Subject || pend.AccountID != "acct-a" {
		t.Fatalf("pending: %+v %v %v", pend, ok, err)
	}
	if err := c.FinishApproval(ctx, t1, "acct-a", "link-1"); err != nil {
		t.Fatal(err)
	}
	if err := c.FinishApproval(ctx, t1, "acct-a", "link-1"); err != nil { // idempotent
		t.Fatal(err)
	}
	l, ok, err := c.LinkedConsoleOf(ctx, uid)
	if err != nil || !ok || l.Subject != t1.Subject || l.LinkID != "link-1" || l.AccountID != "acct-a" || l.Unit != UnitWii {
		t.Fatalf("linked: %+v %v %v", l, ok, err)
	}
	if list, err := c.ConsolesOf(ctx, "acct-a"); err != nil || len(list) != 1 {
		t.Fatalf("consoles: %+v %v", list, err)
	}
	if _, _, err := c.BeginApproval(ctx, p.Code, "acct-a", NumberProof{}); err != ErrPairingCode {
		t.Fatalf("single use: %v", err)
	}

	// A second code for a linked console, claimed by another account: refused.
	p2, err := c.IssuePairingCode(ctx, req)
	if err != nil || p2.Code == p.Code {
		t.Fatalf("issue 2: %+v %v", p2, err)
	}
	if _, _, err := c.BeginApproval(ctx, p2.Code, "acct-b", NumberProof{}); err != ErrConsoleLinked {
		t.Fatalf("linked elsewhere: %v", err)
	}
	if _, already, err := c.BeginApproval(ctx, p2.Code, "acct-a", NumberProof{}); err != nil || !already {
		t.Fatalf("already linked to same: %v %v", already, err)
	}

	// Unlink by subject, then a new account may claim with a fresh subject.
	if got, err := c.ForgetSubject(ctx, t1.Subject); err != nil || got != uid {
		t.Fatalf("forget subject: %d %v", got, err)
	}
	t3, _, err := c.BeginApproval(ctx, p2.Code, "acct-b", NumberProof{})
	if err != nil || t3.Subject == t1.Subject {
		t.Fatalf("relink: %+v %v", t3, err)
	}
	if err := c.AbortApproval(ctx, t3.Code, "acct-b"); err != nil {
		t.Fatal(err)
	}
	t4, _, err := c.BeginApproval(ctx, p2.Code, "acct-c", NumberProof{})
	if err != nil || t4.Subject == t3.Subject {
		t.Fatalf("after abort: %+v %v", t4, err)
	}
	_ = c.AbortApproval(ctx, t4.Code, "acct-c")

	// Expiry, stuck approvals.
	_, _ = pool.Exec(ctx, `UPDATE wfc_pairing_codes SET expires_at = now() - interval '5 minutes' WHERE user_id=$1`, uid)
	if _, err := c.LookupPairingCode(ctx, p2.Code, NumberProof{}); err != ErrPairingCode {
		t.Fatalf("expired: %v", err)
	}
	_, _ = pool.Exec(ctx, `UPDATE wfc_pairing_codes SET approving='acct-d', subject='424242.1' WHERE code=$1`, p2.Code)
	if st, err := c.StuckApprovals(ctx); err != nil || len(st) == 0 {
		t.Fatalf("stuck: %+v %v", st, err)
	}

	// Per-console rate limit.
	_, _ = pool.Exec(ctx, `DELETE FROM wfc_pairing_codes WHERE user_id=$1`, uid)
	for i := 0; i < PairingNewPerHour; i++ {
		if _, err := c.IssuePairingCode(ctx, req); err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
		_, _ = pool.Exec(ctx, `UPDATE wfc_pairing_codes SET used_at = now() WHERE user_id=$1 AND used_at IS NULL`, uid)
	}
	if _, err := c.IssuePairingCode(ctx, req); err != ErrPairingLimit {
		t.Fatalf("limit: %v", err)
	}
}

type fakeCore struct {
	linkID, account string
	active          bool
	err             error
	asked           int
}

func (f *fakeCore) LinkActive(ctx context.Context, subject string) (string, string, bool, error) {
	f.asked++
	return f.linkID, f.account, f.active, f.err
}

func TestLinkGate(t *testing.T) {
	c, pool := testConn(t)
	ctx := context.Background()
	const uid = 434343
	clean := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM wfc_pairing_codes WHERE user_id=$1`, uid)
		_ = c.ForgetPairing(ctx, uid)
	}
	clean()
	defer clean()
	core := &fakeCore{}
	if r, err := c.LinkGate(ctx, uid, core); r != GateUnlinked || err != nil || core.asked != 0 {
		t.Fatalf("fresh: %v %v asked %d", r, err, core.asked)
	}
	p, _ := c.IssuePairingCode(ctx, PairingRequest{UserID: uid, Unit: UnitDS})
	tg, _, err := c.BeginApproval(ctx, p.Code, "acct-g", NumberProof{})
	if err != nil {
		t.Fatal(err)
	}
	core.err = context.DeadlineExceeded
	if r, _ := c.LinkGate(ctx, uid, core); r != GateUnavailable {
		t.Fatalf("core down mid-approval: %v", r)
	}
	core.err = nil
	if r, _ := c.LinkGate(ctx, uid, core); r != GateUnlinked {
		t.Fatalf("core says no: %v", r)
	}
	core.linkID, core.account, core.active = "link-g", "acct-g", true
	if r, err := c.LinkGate(ctx, uid, core); r != GateLinked || err != nil {
		t.Fatalf("heal: %v %v", r, err)
	}
	if l, ok, _ := c.LinkedConsoleOf(ctx, uid); !ok || l.Subject != tg.Subject || l.LinkID != "link-g" {
		t.Fatalf("healed row: %+v", l)
	}
	core.err = context.DeadlineExceeded
	if r, _ := c.LinkGate(ctx, uid, core); r != GateLinked {
		t.Fatalf("cached console must pass with core down: %v", r)
	}
}

// TestEraseUsers runs against a real Postgres when WFC_TEST_DSN is set. It needs the users
// table of a WFC database; other tables are skipped when absent.
func TestEraseUsers(t *testing.T) {
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
	const a, b = int64(9000000001), int64(9000000002)
	for _, u := range []int64{a, b} {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE user_id=$1`, u)
		if _, err := pool.Exec(ctx, `INSERT INTO users (user_id, gsbrcd, password, ng_device_id) VALUES ($1,$2,'',$3)`, u, "ERASE"+strings.Repeat("0", 1)+string(rune('A'+u-a)), []int64{u}); err != nil {
			t.Fatal(err)
		}
		_, _ = pool.Exec(ctx, `INSERT INTO wfc_pairing_linked (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, u)
	}
	defer pool.Exec(ctx, `DELETE FROM users WHERE user_id = ANY($1)`, []int64{a, b})
	defer pool.Exec(ctx, `DELETE FROM wfc_pairing_linked WHERE user_id = ANY($1)`, []int64{a, b})

	n, err := c.EraseUsers(ctx, []int64{a})
	if err != nil || n < 2 {
		t.Fatalf("first erase: %d %v", n, err)
	}
	if n, err = c.EraseUsers(ctx, []int64{a}); err != nil || n != 0 {
		t.Fatalf("second erase: %d %v", n, err)
	}
	var left int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE user_id=$1`, b).Scan(&left)
	if left != 1 {
		t.Fatal("other account's user row was touched")
	}
}

func TestConsoleBanned(t *testing.T) {
	c, pool := testConn(t)
	ctx := context.Background()
	const uid, other = int64(9100000001), int64(9100000002)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE user_id = ANY($1)`, []int64{uid, other})
	defer pool.Exec(ctx, `DELETE FROM users WHERE user_id = ANY($1)`, []int64{uid, other})
	if _, err := pool.Exec(ctx, `INSERT INTO users (user_id, gsbrcd, password, ng_device_id, has_ban, ban_tos, ban_reason, last_ip_address)
		VALUES ($1,'RMCJBAN0','',array[77]::bigint[],true,true,'cheating','198.51.100.7')`, other); err != nil {
		t.Fatal(err)
	}
	if b, _, err := c.ConsoleBanned(ctx, uint64(uid), 0, ""); b || err != nil {
		t.Fatalf("clean console: %v %v", b, err)
	}
	if b, r, err := c.ConsoleBanned(ctx, uint64(uid), 77, ""); !b || r != "cheating" || err != nil {
		t.Fatalf("device match: %v %q %v", b, r, err)
	}
	if b, _, _ := c.ConsoleBanned(ctx, uint64(uid), 0, "198.51.100.7"); !b {
		t.Fatal("address match")
	}
	if b, _, _ := c.ConsoleBanned(ctx, uint64(other), 0, ""); !b {
		t.Fatal("own ban")
	}
}
