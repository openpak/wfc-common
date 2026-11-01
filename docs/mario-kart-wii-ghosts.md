# Mario Kart Wii ghost archive

## Tables

- `mario_kart_wii_sake` (one row per course and profile id, the player's best): what the game
  ranks and downloads. Unchanged.
- `mario_kart_wii_ghosts` (additive, created idempotently by `EnsureMarioKartWiiGhostArchive`):
  one row per accepted upload that carried a ghost body, with `ghost` (the RKG as stored, Mii
  creator data cleared) and header metadata: Mii name, vehicle, character, controller, date set,
  ghost type, drift type, lap count and times, country/state/location codes. Unique on
  `(courseid, pid, score, upload_time)`; rows already in `mario_kart_wii_sake` are backfilled
  with the same key at start, so the backfill can run any number of times.

Competition uploads (the `contest` flag) store no body, so they are ranked but have no archive row.

## RKG header (facts)

Parsed by `common.ParseRKGMetadata` from the community-documented layout, big-endian bit fields:
finish time 0x04 (7 bit minutes, 7 bit seconds, 10 bit ms), course 0x07 (6 bits), vehicle 0x08
(6 bits), character 0x08+6 (6 bits), year since 2000 0x09+4 (7), month 0x0A+3 (4), day 0x0A+7 (5),
controller 0x0B+4 (4), compressed 0x0C+4, ghost type 0x0C+7 (7), drift type 0x0D+6, input length
0x0E (16), lap count 0x10, lap times 0x11 (3 bytes each, up to five), country 0x34, state 0x35,
location 0x36, Mii data 0x3C (name = ten UTF-16BE characters at offset 2).

Sources: Wiimm's RKG documentation, wiki.tockdom.com "RKG", WiiLink wfc-server (AGPL-3.0). Facts only.
