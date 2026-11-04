# Changelog

## v0.6.0 - 2026-11-04

- Account erasure (openpak/prds#120): `database.EraseUsers` deletes everything the shared WFC database holds for a set of console user ids in one transaction (profiles in `users`, `gamestats_public_data`, `sake_records` owned by the profiles, `mario_kart_wii_sake` and `mario_kart_wii_ghosts` rows, legacy `sessions`, `events` rows naming the profile, pairing codes and link marks). Idempotent; tables an older deployment lacks are skipped. `service.EraseHandler` serves `DELETE /internal/accounts/{account_id}/data` with `Authorization: Bearer $ACCOUNT_ERASE_KEY` (503 when unset), and `mkwboard.CoreLinks.UserIDs` lists the account's `wfc` links. Decisions: ghosts and sake records are deleted, not credited to a deleted user, because they carry the Mii name and profile id; the leaderboards core owns its own scores.

## v0.5.0 - 2026-11-04

- `database`: pairing codes for linking a Wii or DS to an OpenPak account (`IssuePairingCode`, `LookupPairingCode`, `RedeemPairingCode`, `ForgetPairing`; tables `wfc_pairing_codes`, `wfc_pairing_linked`, created by `UpdateTables`/`EnsurePairing`). Codes are 8 symbols, valid 30 minutes, single use.

## v0.4.0 - 2026-11-04

- `mkwboard`: the Wii/DS adapter over the neutral leaderboards core (gRPC client, console user id to account through the account core's `wfc` links, Mario Kart Wii boards, `Reconcile` import of the game's own time table). See [docs/leaderboards.md](docs/leaderboards.md).
- `database`: `MKWAllTimes`, `MKWUserOfProfile`, `MKWGhostIDs` (read only).

## v0.3.1

- `httplog`: successful `/internal/` requests (service-to-service polls) are counted, not logged one by
  one; one `internal poll: N calls in last minute` line is written per minute. Failures (status >= 400)
  still log individually.

## v0.3.0

- `httplog`: one log line per request (`http <method> <path> -> <status> <duration>ms`, path only, never
  the query, headers or bodies; `/healthz` skipped) and `httplog.Command` for the GameSpy-style TCP/UDP
  protocols (`<proto> cmd <name> -> <result> <duration>ms`, no payload values). Flusher and Hijacker
  are preserved. `service.Server` now wraps its handler in it, so every service on it logs by default.

## v0.2.3

- `wfcapi.Login.UserID` and a trailing `userID` argument on `qr2client.Login`: the console user id
  (the id the account core links) travels with the session projection so qr2 can report online
  sessions to rewards. Additive on the wire; older peers leave it zero.

## v0.2.2

- `MKWRankRow.UserID` (the console the profile belongs to, from `users`) and `database.MKWRankOfUser`
  (a console's best place on a board): the rankings contract identifies Wii players by the console id
  the account core links, not by the game profile id.

## v0.2.1

- `database.MKWRankOf`: a profile's position on a Mario Kart Wii course board, worldwide or one region, in the order `MKWRankings` uses (for the rankings contract `around` slice).

## v0.2.0

- Mario Kart Wii: every uploaded ghost is now also kept in `mario_kart_wii_ghosts` with the
  metadata read from its RKG header (see [docs/mario-kart-wii-ghosts.md](docs/mario-kart-wii-ghosts.md)).
  `mario_kart_wii_sake`, which the game reads, is unchanged and behaves exactly as before.
- `common.ParseRKGMetadata`, `MiiNameFromData`, `FormatMKWTime` and names for courses, vehicles,
  characters and controllers.
- Read queries for the wfc-race read API (`database.MKW*`), and `EnsureMarioKartWiiGhostArchive`
  (idempotent schema plus backfill of existing ghosts; also run by `UpdateTables`).

## v0.1.0

First split release.
