# Wii/DS WFC Common Library

OpenPak's Wii/DS WFC Common Library.

| | |
| --- | --- |
| Network | Wii and Nintendo DS |
| Language | Go module `openpak/wfc-common` (Go 1.25) |
| Based on | [WiiLink WFC](https://github.com/WiiLink24/wfc-server), AGPL-3.0 |
| Licence | AGPL-3.0-only |

## Packages

| Package | What |
| --- | --- |
| `common` | Shared wire types, config, encoding, encryption, friend codes, IP helpers, auth tokens, match commands, Mii and Mario Kart Wii data |
| `database` | Database access: connection, schema, users, login, events, gamestats and SAKE filters |
| `dwc` | DWC auth request parsing and response writing |
| `filter` | Expression scanner, parser and evaluator (modified from goexpression) |
| `gperror` | GameSpy error codes and WFC error messages |
| `logging` | Levelled logging, output selection and event webhooks |
| `gpcmclient` | GPCM client helpers (player search verification) |
| `qr2client` | QR2 client (session server list, group info) |
| `service` | Internal key, guard and proxy HTTP helpers |
| `wfcapi` | Internal API contract: client, request decode and reply, room, player and group types |

## Use

```go
import "openpak/wfc-common/wfcapi"
```

Consumers use a sibling replace and commit `vendor/`; details in [deploy](docs/deploy.md).

## Docs

- [Protocol](docs/protocol.md)
- [Evidence](docs/evidence.md)
- [Deploy](docs/deploy.md)
- [Notes](docs/notes.md)
- [Service split](docs/service-split.md)

## Licence

AGPL-3.0-only. See [LICENSE](LICENSE). Derived from WiiLink WFC, AGPL-3.0.
