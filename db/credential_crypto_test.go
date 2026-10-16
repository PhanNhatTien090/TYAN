package db

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestTaskPasswordEncryption(t *testing.T) {
	t.Setenv("TYAN_CREDENTIAL_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	secret := " sensitive-password "
	sealed, err := sealTaskPassword(secret)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sealed, secret) || !strings.HasPrefix(sealed, credentialPrefix) {
		t.Fatal("password not protected")
	}
	other, _ := sealTaskPassword(secret)
	if other == sealed {
		t.Fatal("nonce reused")
	}
	opened, err := openTaskPassword(sealed)
	if err != nil || opened != secret {
		t.Fatalf("round trip failed: %v", err)
	}
	if _, err = openTaskPassword("legacy-password"); err == nil {
		t.Fatal("accepted plaintext")
	}
	t.Setenv("TYAN_CREDENTIAL_KEY", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32))))
	if _, err = openTaskPassword(sealed); err == nil {
		t.Fatal("accepted wrong key")
	}
	t.Setenv("TYAN_CREDENTIAL_KEY", "")
	if _, err = sealTaskPassword(secret); err == nil {
		t.Fatal("accepted missing key")
	}
	if _, err = sealTaskPassword(""); err != nil {
		t.Fatal("empty password should not require a key")
	}
}
