# Protocol: wfc-common

Nothing recorded yet.

## Service split rules

wfc-common holds the shared wire formats, identity and friend-code helpers, configuration,
database access and authenticated HTTP transport for the split Wii/DS services; it has no
daemon or image. The full split adds immutable GameSpy error/form codecs and versioned wfcapi,
gpcmclient and qr2client contracts. No protocol server implementation or in-process
authoritative session state belongs here. Internal calls use bounded timeouts and fail closed;
every owner authenticates internal requests.
