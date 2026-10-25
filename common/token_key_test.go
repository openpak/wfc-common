package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSharedTokenKeySurvivesProcessReinitialization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token-key")
	if err := os.WriteFile(path, []byte("test-only-shared-wfc-token-material-32-bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WFC_TOKEN_KEY_FILE", path)
	if err := LoadTokenKey(); err != nil {
		t.Fatal(err)
	}
	auth := NASAuthToken{UserID: 1234}
	raw := auth.Marshal()
	ticket := GPCMLoginTicket{ProfileID: 5678}.Marshal()
	authTokenKey = generateRandom(16)
	loginTicketKey = generateRandom(16)
	if err := LoadTokenKey(); err != nil {
		t.Fatal(err)
	}
	var decoded NASAuthToken
	if err := decoded.Unmarshal(raw); err != nil || decoded.UserID != 1234 {
		t.Fatal("cross-process NAS token failed", err)
	}
	var login GPCMLoginTicket
	if err := login.Unmarshal(ticket); err != nil || login.ProfileID != 5678 {
		t.Fatal("cross-process login ticket failed", err)
	}
	if err := os.WriteFile(path, []byte("different-test-only-material-at-least-32-bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadTokenKey(); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Unmarshal(raw); err == nil {
		t.Fatal("wrong key accepted")
	}
	t.Setenv("WFC_TOKEN_KEY_FILE", "")
	if err := LoadTokenKey(); err == nil {
		t.Fatal("missing key accepted")
	}
}
