# Changelog

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
