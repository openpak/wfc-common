# Deploy: wfc-common

All processes call common.LoadTokenKey before serving; WFC_TOKEN_KEY_FILE names
an operator-generated binary key of at least 32 bytes. Purpose-derived keys preserve
the native NAS and login-ticket wire formats across independent process restarts.
Existing process-random tickets require a reconnect at initial migration.
Internal HTTP requests use WFC_INTERNAL_KEY, at least 32 bytes, and fail closed.

Consumers use a sibling replace and commit vendor/, so their standalone builds
need no workspace or sibling checkout. Refresh all consumers after changes.
`GOWORK=off go test ./...` and `GOWORK=off go vet ./...` verify this module.
