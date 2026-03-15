package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func newTestRegistry(t *testing.T, adminKey string) *PlayerRegistry {
	t.Helper()
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")
	return &PlayerRegistry{
		accounts: make(map[string]*PlayerAccount),
		keys:     make(map[string]*PlayerAccount),
		adminKey: adminKey,
		filePath: fp,
	}
}

// --- Registration ---

func TestRegisterBasic(t *testing.T) {
	pr := newTestRegistry(t, "")
	acc, err := pr.Register("Alice", "pass1234")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if acc.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", acc.Name)
	}
	if !strings.HasPrefix(acc.APIKey, "xk-") {
		t.Errorf("expected API key prefix xk-, got %s", acc.APIKey)
	}
	if acc.PasswordHash == "" {
		t.Error("password hash should not be empty")
	}
	if acc.PasswordHash == "pass1234" {
		t.Error("password hash should not be plaintext")
	}
}

func TestRegisterDuplicateName(t *testing.T) {
	pr := newTestRegistry(t, "")
	_, err := pr.Register("Alice", "pass1234")
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	_, err = pr.Register("alice", "otherpass")
	if err == nil {
		t.Fatal("expected error for duplicate name (case-insensitive)")
	}
	if !strings.Contains(err.Error(), "already taken") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegisterNameValidation(t *testing.T) {
	pr := newTestRegistry(t, "")

	tests := []struct {
		name string
		pass string
		want string
	}{
		{"", "pass1234", "2-24 characters"},
		{"A", "pass1234", "2-24 characters"},
		{"ABCDEFGHIJKLMNOPQRSTUVWXY", "pass1234", "2-24 characters"}, // 25 chars
		{"  ", "pass1234", "2-24 characters"},                        // trimmed to empty
		{"Alice", "abc", "at least 4"},
		{"Alice", "", "at least 4"},
	}

	for _, tt := range tests {
		_, err := pr.Register(tt.name, tt.pass)
		if err == nil {
			t.Errorf("Register(%q, %q) should have failed", tt.name, tt.pass)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Register(%q, %q) error = %q, want substring %q", tt.name, tt.pass, err.Error(), tt.want)
		}
	}
}

func TestRegisterUniqueAPIKeys(t *testing.T) {
	pr := newTestRegistry(t, "")
	acc1, _ := pr.Register("Alice", "pass1234")
	acc2, _ := pr.Register("Bob", "pass5678")
	if acc1.APIKey == acc2.APIKey {
		t.Error("two accounts should not share an API key")
	}
}

// --- Login ---

func TestLoginSuccess(t *testing.T) {
	pr := newTestRegistry(t, "")
	_, err := pr.Register("Alice", "pass1234")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	acc, err := pr.Login("Alice", "pass1234")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if acc.Name != "Alice" {
		t.Errorf("expected Alice, got %s", acc.Name)
	}
}

func TestLoginCaseInsensitive(t *testing.T) {
	pr := newTestRegistry(t, "")
	pr.Register("Alice", "pass1234")
	_, err := pr.Login("ALICE", "pass1234")
	if err != nil {
		t.Errorf("login should be case-insensitive for name: %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	pr := newTestRegistry(t, "")
	pr.Register("Alice", "pass1234")
	_, err := pr.Login("Alice", "wrong")
	if err == nil {
		t.Fatal("login with wrong password should fail")
	}
	if !strings.Contains(err.Error(), "wrong password") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoginUnknownPlayer(t *testing.T) {
	pr := newTestRegistry(t, "")
	_, err := pr.Login("Nobody", "pass1234")
	if err == nil {
		t.Fatal("login for unknown player should fail")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Authenticate ---

func TestAuthenticatePlayerKey(t *testing.T) {
	pr := newTestRegistry(t, "")
	acc, _ := pr.Register("Alice", "pass1234")

	name, admin, ok := pr.Authenticate(acc.APIKey)
	if !ok {
		t.Fatal("authenticate should succeed with valid player key")
	}
	if admin {
		t.Error("player key should not be admin")
	}
	if name != "Alice" {
		t.Errorf("expected player name Alice, got %s", name)
	}
}

func TestAuthenticateAdminKey(t *testing.T) {
	pr := newTestRegistry(t, "my-admin-key")

	name, admin, ok := pr.Authenticate("my-admin-key")
	if !ok {
		t.Fatal("authenticate should succeed with admin key")
	}
	if !admin {
		t.Error("admin key should return admin=true")
	}
	if name != "" {
		t.Errorf("admin key should return empty player name, got %s", name)
	}
}

func TestAuthenticateInvalidKey(t *testing.T) {
	pr := newTestRegistry(t, "admin-key")
	pr.Register("Alice", "pass1234")

	_, _, ok := pr.Authenticate("bogus-key")
	if ok {
		t.Error("authenticate should fail with invalid key")
	}
}

func TestAuthenticateEmptyKey(t *testing.T) {
	pr := newTestRegistry(t, "admin-key")
	_, _, ok := pr.Authenticate("")
	if ok {
		t.Error("authenticate should fail with empty key")
	}
}

func TestAuthenticateNoAdminKey(t *testing.T) {
	pr := newTestRegistry(t, "")
	// With no admin key configured, only player keys should work
	_, _, ok := pr.Authenticate("anything")
	if ok {
		t.Error("should not authenticate random key when no admin key is set")
	}
}

// --- GetAccount ---

func TestGetAccount(t *testing.T) {
	pr := newTestRegistry(t, "")
	pr.Register("Alice", "pass1234")

	acc := pr.GetAccount("alice")
	if acc == nil {
		t.Fatal("GetAccount should find alice (case-insensitive)")
	}
	if acc.Name != "Alice" {
		t.Errorf("expected Alice, got %s", acc.Name)
	}

	acc = pr.GetAccount("nobody")
	if acc != nil {
		t.Error("GetAccount should return nil for unknown name")
	}
}

// --- Persistence ---

func TestPersistenceSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	// Create registry and register accounts
	pr1 := &PlayerRegistry{
		accounts: make(map[string]*PlayerAccount),
		keys:     make(map[string]*PlayerAccount),
		adminKey: "admin",
		filePath: fp,
	}
	acc1, _ := pr1.Register("Alice", "pass1234")
	acc1.PlayerID = 1
	acc2, _ := pr1.Register("Bob", "pass5678")
	acc2.PlayerID = 2
	pr1.saveLocked() // save again to capture PlayerID updates

	// Create new registry from same file
	pr2 := &PlayerRegistry{
		accounts: make(map[string]*PlayerAccount),
		keys:     make(map[string]*PlayerAccount),
		adminKey: "admin",
		filePath: fp,
	}
	pr2.load()

	// Verify accounts loaded
	if len(pr2.accounts) != 2 {
		t.Fatalf("expected 2 accounts after load, got %d", len(pr2.accounts))
	}

	// Verify login works with loaded accounts
	loadedAcc, err := pr2.Login("Alice", "pass1234")
	if err != nil {
		t.Fatalf("login after load failed: %v", err)
	}
	if loadedAcc.PlayerID != 1 {
		t.Errorf("expected PlayerID 1, got %d", loadedAcc.PlayerID)
	}

	// Verify API key auth works after load
	name, _, ok := pr2.Authenticate(acc1.APIKey)
	if !ok {
		t.Fatal("API key auth should work after load")
	}
	if name != "Alice" {
		t.Errorf("expected Alice, got %s", name)
	}

	// Verify second account too
	name, _, ok = pr2.Authenticate(acc2.APIKey)
	if !ok {
		t.Fatal("Bob's API key should work after load")
	}
	if name != "Bob" {
		t.Errorf("expected Bob, got %s", name)
	}
}

func TestPersistenceFileFormat(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	pr := &PlayerRegistry{
		accounts: make(map[string]*PlayerAccount),
		keys:     make(map[string]*PlayerAccount),
		filePath: fp,
	}
	pr.Register("Alice", "pass1234")

	// Read the file and verify it's valid JSON with password hash
	data, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("failed to read accounts file: %v", err)
	}

	var saved []savedAccount
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("accounts file is not valid JSON: %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("expected 1 saved account, got %d", len(saved))
	}
	if saved[0].Name != "Alice" {
		t.Errorf("expected name Alice, got %s", saved[0].Name)
	}
	if saved[0].PasswordHash == "" {
		t.Error("password hash should be persisted")
	}
	if !strings.HasPrefix(saved[0].APIKey, "xk-") {
		t.Errorf("expected API key prefix xk-, got %s", saved[0].APIKey)
	}
}

func TestPersistenceFilePermissions(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	pr := &PlayerRegistry{
		accounts: make(map[string]*PlayerAccount),
		keys:     make(map[string]*PlayerAccount),
		filePath: fp,
	}
	pr.Register("Alice", "pass1234")

	info, err := os.Stat(fp)
	if err != nil {
		t.Fatalf("failed to stat accounts file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected file permissions 0600, got %04o", perm)
	}
}

func TestPersistenceMissingFile(t *testing.T) {
	pr := &PlayerRegistry{
		accounts: make(map[string]*PlayerAccount),
		keys:     make(map[string]*PlayerAccount),
		filePath: filepath.Join(t.TempDir(), "nonexistent.json"),
	}
	pr.load() // should not panic or error
	if len(pr.accounts) != 0 {
		t.Error("loading nonexistent file should result in empty accounts")
	}
}

func TestPersistenceAutoSaveOnRegister(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	pr := &PlayerRegistry{
		accounts: make(map[string]*PlayerAccount),
		keys:     make(map[string]*PlayerAccount),
		filePath: fp,
	}
	pr.Register("Alice", "pass1234")

	// File should exist without explicit save call
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		t.Error("accounts file should be auto-created on register")
	}
}

// --- Concurrency ---

func TestConcurrentRegisterAndAuth(t *testing.T) {
	pr := newTestRegistry(t, "admin")
	var wg sync.WaitGroup

	// Register 20 accounts concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := strings.Repeat("u", 2) + string(rune('a'+n))
			pr.Register(name, "pass1234")
		}(i)
	}
	wg.Wait()

	// All should be authenticatable
	pr.mu.RLock()
	keyCount := len(pr.keys)
	pr.mu.RUnlock()

	if keyCount != 20 {
		t.Errorf("expected 20 registered keys, got %d", keyCount)
	}

	// Concurrent auth
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pr.Authenticate("admin")
		}()
	}
	wg.Wait()
}
