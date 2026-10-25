# Notes: wfc-common

Shared wire formats, identity and friend-code helpers, configuration, database
access and authenticated HTTP transport for the split Wii/DS services.
No daemon or container image. AGPL-3.0, derived from WiiLink WFC.

Full protocol split adds immutable GameSpy error/form codecs and versioned
wfcapi, gpcmclient and qr2client contracts. No protocol server implementation or
in-process authoritative session state belongs in this module. Internal calls
use bounded timeouts and fail closed; all owners authenticate internal requests.
