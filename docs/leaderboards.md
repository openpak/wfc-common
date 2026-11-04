# Leaderboards (Mario Kart Wii)

The Mario Kart Wii times are stored by the neutral leaderboards core, not by the WFC database. Package
`mkwboard` is the adapter, used by wfc-sake (writes) and wfc-race (reads).

| Core concept | Mario Kart Wii |
| --- | --- |
| network / title | `wfc` / `1687` (GameSpy game id), mapped to the neutral title `mario-kart-wii` |
| board | course id (`0`-`31`, competitions up to `32767`) worldwide; `r<region>-<course>` for one region |
| account | the account core's `wfc` link: subject = `users.user_id` (the console) |
| aggregation | `latest`, ascending: every accepted upload replaces the player's time |
| columns / info | `pid`, `uid`, `region` / the Mii player info as uploaded |

A console that is not linked to an account has no entry: its time stays in `mario_kart_wii_sake`
(which remains the ghost-file store) until the link exists, then `Reconcile` (wfc-sake, at start and
every ten minutes) imports it. `Reconcile` never changes the table and never overwrites a newer core time.

Environment (wfc-sake, wfc-race): `WFC_LEADERBOARDS_ADDR` (`openpak-leaderboards:20211`),
`WFC_LEADERBOARDS_KEY`, `WFC_CORE_ADDRESS` (`openpak-account:20001`), `WFC_CORE_KEY`.
