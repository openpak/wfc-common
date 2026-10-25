# wfc-common

Shared wire formats, identity and friend-code helpers, configuration, database
access and authenticated HTTP transport for the split Wii/DS services.
No daemon or container image. AGPL-3.0, derived from WiiLink WFC.

Consumers use a sibling replace and commit vendor/, so their standalone builds
need no workspace or sibling checkout. Refresh all consumers after changes.
`GOWORK=off go test ./...` and `GOWORK=off go vet ./...` verify this module.

All processes call common.LoadTokenKey before serving; WFC_TOKEN_KEY_FILE names
an operator-generated binary key of at least 32 bytes. Purpose-derived keys preserve
the native NAS and login-ticket wire formats across independent process restarts.
Existing process-random tickets require a reconnect at initial migration.
Internal HTTP requests use WFC_INTERNAL_KEY, at least 32 bytes, and fail closed.

Full protocol split adds immutable GameSpy error/form codecs and versioned
wfcapi, gpcmclient and qr2client contracts. No protocol server implementation or
in-process authoritative session state belongs in this module. Internal calls
use bounded timeouts and fail closed; all owners authenticate internal requests.
