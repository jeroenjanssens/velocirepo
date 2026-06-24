package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

func TestParseEnvFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte(`# comment
PLAUSIBLE_API_KEY=abc123
GITHUB_TOKEN="ghp_secret"
YOUTUBE_TOKEN='yt_key'

EMPTY=
`), 0644)

	secrets, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatal(err)
	}

	if secrets["PLAUSIBLE_API_KEY"] != "abc123" {
		t.Errorf("PLAUSIBLE_API_KEY = %q, want abc123", secrets["PLAUSIBLE_API_KEY"])
	}
	if secrets["GITHUB_TOKEN"] != "ghp_secret" {
		t.Errorf("GITHUB_TOKEN = %q, want ghp_secret", secrets["GITHUB_TOKEN"])
	}
	if secrets["YOUTUBE_TOKEN"] != "yt_key" {
		t.Errorf("YOUTUBE_TOKEN = %q, want yt_key", secrets["YOUTUBE_TOKEN"])
	}
	if secrets["EMPTY"] != "" {
		t.Errorf("EMPTY = %q, want empty", secrets["EMPTY"])
	}
}

func TestEncryptSecret(t *testing.T) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := encryptSecret(pub[:], "hello")
	if err != nil {
		t.Fatal(err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, ok := box.OpenAnonymous(nil, ciphertext, pub, priv)
	if !ok {
		t.Fatal("failed to decrypt")
	}
	if string(decrypted) != "hello" {
		t.Errorf("decrypted = %q, want hello", string(decrypted))
	}
}

func TestCiSyncSecretsDryRun(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"
`), 0644)

	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("MY_SECRET=value123\n"), 0644)

	_, buf, err := execCmd(cfgPath, "sync-secrets", "--dry-run", "--env-file", envPath, "--repo", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "MY_SECRET") {
		t.Errorf("output missing MY_SECRET: %s", output)
	}
	if !strings.Contains(output, "Would sync 1 secret") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestCiSyncSecretsDryRunRenamesGitHub(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"
`), 0644)

	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("GITHUB_TOKEN=ghp_secret\nPLAUSIBLE_TOKEN=abc\n"), 0644)

	_, buf, err := execCmd(cfgPath, "sync-secrets", "--dry-run", "--env-file", envPath, "--repo", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "GH_TOKEN") {
		t.Errorf("expected GITHUB_TOKEN renamed to GH_TOKEN: %s", output)
	}
	if strings.Contains(output, "GITHUB_TOKEN") {
		t.Errorf("GITHUB_TOKEN should have been renamed: %s", output)
	}
	if !strings.Contains(output, "Would sync 2 secret") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestCiSyncSecretsIntegration(t *testing.T) {
	var privateKey [32]byte
	rand.Read(privateKey[:])
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	secretsSet := make(map[string]string)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/actions/secrets/public-key" {
			json.NewEncoder(w).Encode(map[string]string{
				"key":    base64.StdEncoding.EncodeToString(publicKey[:]),
				"key_id": "key-123",
			})
			return
		}

		if r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/repos/owner/repo/actions/secrets/") {
			name := strings.TrimPrefix(r.URL.Path, "/repos/owner/repo/actions/secrets/")
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			secretsSet[name] = body["encrypted_value"]
			w.WriteHeader(http.StatusCreated)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	oldBase := githubAPIBase
	githubAPIBase = server.URL
	defer func() { githubAPIBase = oldBase }()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"
`), 0644)

	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("PLAUSIBLE_API_KEY=test123\nYOUTUBE_TOKEN=yt456\n"), 0644)

	t.Setenv("GITHUB_TOKEN", "fake-token")

	_, buf, err := execCmd(cfgPath, "sync-secrets", "--force", "--env-file", envPath, "--repo", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "Synced 2 secret") {
		t.Errorf("unexpected output: %s", output)
	}

	if _, ok := secretsSet["PLAUSIBLE_API_KEY"]; !ok {
		t.Error("PLAUSIBLE_API_KEY not set")
	}
	if _, ok := secretsSet["YOUTUBE_TOKEN"]; !ok {
		t.Error("YOUTUBE_TOKEN not set")
	}

	// Verify we can decrypt
	for name, enc := range secretsSet {
		ciphertext, err := base64.StdEncoding.DecodeString(enc)
		if err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		_, ok := box.OpenAnonymous(nil, ciphertext, &publicKey, &privateKey)
		if !ok {
			t.Errorf("failed to decrypt %s", name)
		}
	}
}
