# Account erasure

Every WFC service shares one Postgres database, so one call erases an account for all of
them. wfc-api serves it (see its docs); the logic lives here:

- `database.EraseUsers(ctx, userIDs)` - one transaction, idempotent, returns the row count.
- `service.EraseHandler(name, fn)` - `DELETE /internal/accounts/{account_id}/data`, bearer
  `ACCOUNT_ERASE_KEY` (endpoint answers 503 when the variable is unset).
- `mkwboard.CoreLinks.UserIDs(ctx, account)` - the account's `wfc` links (console user ids).

Erased: `users` rows (all game profiles of the console), `gamestats_public_data`, `sake_records`
owned by those profiles, `mario_kart_wii_sake`, `mario_kart_wii_ghosts`, `sessions`, `events`
rows whose `profile_id` is one of them, `wfc_pairing_codes`, `wfc_pairing_linked`.
Kept: nothing authored by the account; leaderboard scores live in the leaderboards core and
are erased by its own step. Wii/DS buddy lists are not stored server-side.

Since v0.7.0 the `users` row holds protocol ids (profile_id, user_id, gsbrcd, ng_device_id, password), the ban fields,
`firstname`/`lastname` (game-written) and a last address that is blanked after 90 days. No email or nick column exists:
both are derived from user_id and gsbrcd. Names for a console's player belong to the linked core account (pairing flow);
replacing `firstname`/`lastname` with it needs a decision on which titles read them back, so they stay for now.

Link-first (v0.8.0, prds#139): `wfc_pairing_linked` also caches the console's core subject
(`<user_id>.<unix ns>`), link id, account id and unit, and pairing codes carry the game and
in-game name the console sent. Both are erased with the console. wfc-api's erase reads the
console ids from the core's links and from this cache.
