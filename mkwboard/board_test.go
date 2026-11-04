package mkwboard

import (
	"context"
	"testing"
	"time"

	"openpak/wfc-common/mkwboard/fakeleaderboards"
)

type links map[int64]string

func (l links) Account(_ context.Context, uid int64) (string, bool, error) {
	a, ok := l[uid]
	return a, ok, nil
}

type src []Time

func (s src) AllTimes(context.Context) ([]Time, error) { return s, nil }

const acctA, acctB = "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"

func newBoard(t *testing.T, f *fakeleaderboards.Fake, l Linker) *Board {
	t.Helper()
	cli, conn, err := Dial("passthrough:///bufnet", "k", f.Serve(t)...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return &Board{LB: cli, Links: l}
}

func TestSubmitRangeAndUnlinked(t *testing.T) {
	f := fakeleaderboards.New("k")
	b := newBoard(t, f, links{1: acctA, 2: acctB})
	ctx := context.Background()
	if err := b.Declare(ctx); err != nil || len(f.Mapped) != 1 || f.Synced != Courses {
		t.Fatalf("declare: %v %v %d", err, f.Mapped, f.Synced)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(b.Submit(ctx, Time{UserID: 1, PID: 10, Region: 3, Course: 4, Score: 90000, PlayerInfo: "aaa"}))
	must(b.Submit(ctx, Time{UserID: 2, PID: 20, Region: 3, Course: 4, Score: 80000, PlayerInfo: "bbb"}))
	if err := b.Submit(ctx, Time{UserID: 9, PID: 90, Course: 4, Score: 1}); err != ErrNoLink {
		t.Fatalf("unlinked: %v", err)
	}
	// An upload replaces the player's previous time even when slower.
	must(b.Submit(ctx, Time{UserID: 2, PID: 20, Region: 3, Course: 4, Score: 95000, PlayerInfo: "bbb"}))
	got, total, err := b.Range(ctx, 0, 4, 0, 10)
	must(err)
	if total != 2 || len(got) != 2 || got[0].PID != 10 || got[0].Score != 90000 || got[1].Score != 95000 || got[1].Region != 3 || got[0].PlayerInfo != "aaa" || got[0].UserID != 1 {
		t.Fatalf("worldwide %+v total %d", got, total)
	}
	reg, _, _ := b.Range(ctx, 3, 4, 0, 10)
	other, _, _ := b.Range(ctx, 5, 4, 0, 10)
	if len(reg) != 2 || len(other) != 0 {
		t.Fatalf("regional %d %d", len(reg), len(other))
	}
}

// Restart: the board lives in the core, so a new Board over the same core sees the same times.
func TestRestartKeepsTimes(t *testing.T) {
	f := fakeleaderboards.New("k")
	ctx := context.Background()
	b1 := newBoard(t, f, links{1: acctA})
	if err := b1.Submit(ctx, Time{UserID: 1, PID: 10, Course: 0, Score: 70000, PlayerInfo: "x"}); err != nil {
		t.Fatal(err)
	}
	b2 := newBoard(t, f, links{1: acctA}) // a fresh process
	got, total, err := b2.Range(ctx, 0, 0, 0, 10)
	if err != nil || total != 1 || got[0].Score != 70000 {
		t.Fatalf("after restart: %+v %d %v", got, total, err)
	}
}

func TestReconcileIsIdempotent(t *testing.T) {
	f := fakeleaderboards.New("k")
	ctx := context.Background()
	b := newBoard(t, f, links{1: acctA, 2: acctB})
	s := src{
		{UserID: 1, PID: 10, Region: 2, Course: 1, Score: 61000, PlayerInfo: "a"},
		{UserID: 2, PID: 20, Region: 0 + 2, Course: 1, Score: 62000, PlayerInfo: "b"},
		{UserID: 1, PID: 10, Region: 2, Course: 7, Score: 70000, PlayerInfo: "a"},
		{UserID: 5, PID: 50, Region: 2, Course: 7, Score: 71000, PlayerInfo: "c"}, // not linked
	}
	st, err := b.Reconcile(ctx, s)
	if err != nil || st != (Stats{Seen: 4, Imported: 3, Unlinked: 1}) {
		t.Fatalf("first run %+v %v", st, err)
	}
	st, err = b.Reconcile(ctx, s)
	if err != nil || st != (Stats{Seen: 4, Present: 3, Unlinked: 1}) {
		t.Fatalf("second run %+v %v", st, err)
	}
	if n := f.Entries("1687", "1"); n != 2 {
		t.Fatalf("entries %d", n)
	}
	// A newer core time is not overwritten by the older table row.
	if err := b.Submit(ctx, Time{UserID: 1, PID: 10, Region: 2, Course: 1, Score: 59000}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Reconcile(ctx, s); err != nil {
		t.Fatal(err)
	}
	got, _, _ := b.OfAccounts(ctx, 0, 1, []string{acctA})
	if len(got) != 1 || got[0].Score != 59000 {
		t.Fatalf("overwritten: %+v", got)
	}
	// A table row uploaded after the core's entry (a submit lost while the core was down) wins.
	s[0].At, s[0].Score = time.Now().Add(time.Hour), 58000
	if st, _ := b.Reconcile(ctx, s); st.Imported != 1 {
		t.Fatalf("late upload not imported: %+v", st)
	}
	got, _, _ = b.OfAccounts(ctx, 0, 1, []string{acctA})
	if got[0].Score != 58000 {
		t.Fatalf("late upload: %+v", got)
	}
}
