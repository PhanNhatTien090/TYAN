package agent

import (
	"encoding/json"
	"github.com/Autumn-27/artex/db"
	"strings"
	"testing"
)

func TestCredentialMetadataNeverContainsSecrets(t *testing.T) {
	rows := []db.TaskCredential{{ID: 1, Domain: "localhost", Username: "lab-user", Password: "SECRET_PASSWORD_SENTINEL", Note: "SECRET_NOTE_SENTINEL", Source: "user"}}
	b, err := json.Marshal(credentialMetadata(rows))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "SECRET_") {
		t.Fatal("credential secret reached model metadata")
	}
	if !strings.Contains(string(b), `"has_password":true`) {
		t.Fatal("missing operator-action metadata")
	}
}
