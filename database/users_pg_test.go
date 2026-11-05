package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const legacyUsers = `
DROP TABLE IF EXISTS users;
CREATE TABLE users (
 profile_id bigserial PRIMARY KEY, user_id bigint NOT NULL, gsbrcd varchar NOT NULL, password varchar NOT NULL,
 ng_device_id bigint[], email varchar NOT NULL, unique_nick varchar NOT NULL, firstname varchar,
 lastname varchar DEFAULT '', last_ip_address varchar DEFAULT '', last_ingamesn varchar DEFAULT '',
 has_ban boolean DEFAULT false, ban_issued timestamp, ban_expires timestamp, ban_reason varchar,
 ban_reason_hidden varchar, ban_moderator varchar, ban_tos boolean, open_host boolean DEFAULT false,
 allow_default_keys boolean DEFAULT false)`

// Against a real Postgres (WFC_TEST_DSN): the legacy table loses email and unique_nick, a console
// is created and read back with the derived values, and old addresses are blanked.
func TestUsersKeepNoIdentityCopy(t *testing.T) {
	dsn := os.Getenv("WFC_TEST_DSN")
	if dsn == "" {
		t.Skip("WFC_TEST_DSN not set")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, legacyUsers); err != nil {
		t.Fatal(err)
	}
	c := &Connection{pool: pool, ctx: ctx}
	c.UpdateTables()

	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_name='users' AND column_name IN ('email','unique_nick')`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d identity columns remain", n)
	}

	u, err := c.LoginUserToGPCM(4242, "RMCE", 0, false, 0x1234, "203.0.113.9", "sn", true)
	if err != nil || !u.Created {
		t.Fatalf("create: %v %+v", err, u)
	}
	if u.UniqueNick != DerivedNick(4242, "RMCE") || u.Email != DerivedEmail(4242, "RMCE") {
		t.Fatalf("created user: %q %q", u.UniqueNick, u.Email)
	}
	again, err := c.LoginUserToGPCM(4242, "RMCE", 0, false, 0x1234, "203.0.113.9", "sn", true)
	if err != nil || again.ProfileId != u.ProfileId || again.UniqueNick != u.UniqueNick {
		t.Fatalf("second login: %v %+v", err, again)
	}
	if p, ok := c.GetProfile(u.ProfileId); !ok || p.UniqueNick != u.UniqueNick || p.Email != u.Email {
		t.Fatalf("GetProfile: %v %+v", ok, p)
	}
	if gs, err := c.LoginUserToGameStats(4242, "RMCE"); err != nil || gs.UniqueNick != u.UniqueNick {
		t.Fatalf("gamestats: %v %+v", err, gs)
	}

	// Retention: recent addresses stay, old ones go.
	if got, _ := c.PurgeOldIPs(time.Now()); got != 0 {
		t.Fatalf("recent address purged: %d", got)
	}
	pool.Exec(ctx, `UPDATE users SET last_ip_at = now() - interval '91 days'`)
	if got, err := c.PurgeOldIPs(time.Now()); err != nil || got != 1 {
		t.Fatalf("purge: %d %v", got, err)
	}
	var ip string
	pool.QueryRow(ctx, `SELECT last_ip_address FROM users`).Scan(&ip)
	if ip != "" {
		t.Fatal("address kept past retention")
	}
	if p, ok := c.ClearProfile(u.ProfileId); !ok || p.UniqueNick != u.UniqueNick {
		t.Fatalf("ClearProfile: %v %+v", ok, p)
	}
}
