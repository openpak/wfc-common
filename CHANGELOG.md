# Changelog

## v0.8.1 - 2026-11-05

- `ClearConsoleData`: an unlink's "also erase" removes the console's game data and personal fields but keeps its protocol ids. Deleting the profile rows (as `EraseUsers` does for a deleted account) left a DS whose game card still holds the old profile id unable to go online (error 60000) after relinking.

## v0.8.0 - 2026-11-05

- Link-first (prds#139, design `prds/repos/wfc-link-first-design.md`): a console plays online only once it is linked to an OpenPak account.
  - `database` pairing v2: codes live 15 minutes and carry a link number, a NAS return code 4000-9999 that every DWC title shows as "Error Code: 2xxxx" (`DisplayNumber`; measured on Marina and melonDS, step 0 of prds#139), unique among live codes, plus the unit, game, in-game name and a network tag. Limits: 6 new codes per console per hour, 2000 live codes. `ParsePairingInput` takes the 8-symbol code or the 5-digit number the console showed.
  - A claim by number needs a `NumberProof`: the claimer's `NetTags` (keyed, day-scoped hash of the IPv4 address or IPv6 /64, `NetTagKey` = `WFC_NET_TAG_KEY` or `WFC_INTERNAL_KEY`; no address is stored) matching the console's, or the console's in-game name. Otherwise it reads as unknown. `ConsoleBanned` checks a console that has no profile yet.
  - Adapter-owned approval: `BeginApproval` (claims a code for an account, gives it a fresh core subject `<user_id>.<unix ns>` so a console can be relinked by another account), `FinishApproval`, `AbortApproval`, `StuckApprovals`, `PendingApproval`. `RedeemPairingCode` is gone.
  - `wfc_pairing_linked` now caches subject, link id, account and unit; `LinkedConsoleOf`, `ConsolesOf`, `ForgetSubject`, `ForgetAccount`, `PairingCounters`, `CodeHash` (salted tag for logs; codes are never logged).
  - `LinkFirst()` (`WFC_LINK_FIRST=1`), `LinkGate` (cache first; the core is asked only to heal a half-done approval), `LinkedProfile` and `LinkedUser` (60 s positive cache; always true with the gate off).
  - `gperror`: 22100 not linked (Mario Kart Wii text with code and number; other Wii titles get the shown number as wl:err), 22101 unlinked while online, 22102 link state unavailable, in seven languages; `NotLinkedError`, `UnlinkedKickError`, `LinkUnavailableError`.
  - `mkwboard`: `CoreLinks.LinkActive`/`Reserve`/`Activate`/`Unlink`; `UserIDs` reads both subject forms; `CachedLinker` answers from the local link cache first.
  - `wfcapi.Login.Unlinked` and `qr2client.LoginAs` for the Wii DNS-exploit login; `wfcapi.GPCMRequest.UserID` and `gpcmclient.KickUser` (op `kick-user`); `Connection.Ready`.
- Fix: a Mario Kart Wii custom error in a language without a translation (PAL English, Korean) now falls back to English instead of showing no text.

## v0.7.0 - 2026-11-05

- prds#131: a console's row keeps protocol ids only. `users.email` and `users.unique_nick` were functions of (user_id, gsbrcd) (`<base32 id><gsbrcd>[@nds]`), never a person's address; they are computed (`DerivedNick`, `DerivedEmail`) and the columns are dropped at start-up by `UpdateTables` (idempotent). Deploy every service that reads `users` (wfc-gpcm, wfc-api) on this version before any runs `UpdateTables`.
- `last_ip_address` stays for ban matching and now has a retention: `last_ip_at` is stamped at each login and `runIPRetention` (started by `UpdateTables`) blanks addresses unseen for 90 days, every 6 hours.
- `GetProfile`/`ClearProfile` no longer fail on a NULL first or last name.
- Still local, by decision pending: `firstname`/`lastname` (written by the game with `updatepro`; the pairing flow gives no name to replace them, see docs/account-erasure.md).

## v0.6.1 - 2026-11-04

- Fix: core calls now also send `x-internal-key`; the core only reads that header or a Bearer token, so `X-API-Key` alone was rejected as Unauthenticated (found while enabling the erase on the live box).

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
