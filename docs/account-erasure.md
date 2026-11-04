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
