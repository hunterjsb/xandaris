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
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		adminKey:   adminKey,
		filePath:   fp,
	}
}

// --- FindOrCreateByDiscord ---

func TestFindOrCreate_NewAccount(t *testing.T) {
	pr := newTestRegistry(t, "")
	acc, isNew, err := pr.FindOrCreateByDiscord("123456", "Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isNew {
		t.Error("expected new account")
	}
	if acc.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", acc.Name)
	}
	if acc.DiscordID != "123456" {
		t.Errorf("expected discord ID 123456, got %s", acc.DiscordID)
	}
	if !strings.HasPrefix(acc.APIKey, "xk-") {
		t.Errorf("expected API key prefix xk-, got %s", acc.APIKey)
	}
}

func TestFindOrCreate_ExistingAccount(t *testing.T) {
	pr := newTestRegistry(t, "")
	acc1, _, _ := pr.FindOrCreateByDiscord("123456", "Alice")
	acc2, isNew, _ := pr.FindOrCreateByDiscord("123456", "Alice")

	if isNew {
		t.Error("second call should not create new account")
	}
	if acc1.APIKey != acc2.APIKey {
		t.Error("should return same account")
	}
}

func TestFindOrCreate_UpdatesUsername(t *testing.T) {
	pr := newTestRegistry(t, "")
	pr.FindOrCreateByDiscord("123456", "OldName")
	acc, isNew, _ := pr.FindOrCreateByDiscord("123456", "NewName")

	if isNew {
		t.Error("should not be new")
	}
	if acc.Name != "NewName" {
		t.Errorf("expected name NewName, got %s", acc.Name)
	}
	// Old name lookup should be gone
	if pr.GetAccount("OldName") != nil {
		t.Error("old name should be removed from index")
	}
	// New name lookup should work
	if pr.GetAccount("NewName") == nil {
		t.Error("new name should be indexed")
	}
}

func TestFindOrCreate_EmptyDiscordID(t *testing.T) {
	pr := newTestRegistry(t, "")
	_, _, err := pr.FindOrCreateByDiscord("", "Alice")
	if err == nil {
		t.Error("should reject empty discord ID")
	}
}

func TestFindOrCreate_LongUsername(t *testing.T) {
	pr := newTestRegistry(t, "")
	longName := strings.Repeat("a", 50)
	acc, _, _ := pr.FindOrCreateByDiscord("123", longName)
	if len(acc.Name) > 24 {
		t.Errorf("name should be truncated to 24 chars, got %d", len(acc.Name))
	}
}

func TestFindOrCreate_UniqueAPIKeys(t *testing.T) {
	pr := newTestRegistry(t, "")
	acc1, _, _ := pr.FindOrCreateByDiscord("111", "Alice")
	acc2, _, _ := pr.FindOrCreateByDiscord("222", "Bob")
	if acc1.APIKey == acc2.APIKey {
		t.Error("two accounts should not share an API key")
	}
}

func TestFindOrCreate_DifferentDiscordSameName(t *testing.T) {
	pr := newTestRegistry(t, "")
	acc1, _, _ := pr.FindOrCreateByDiscord("111", "Alice")
	acc2, _, _ := pr.FindOrCreateByDiscord("222", "Alice")
	// Both should exist — different Discord users can have same display name
	if acc1.APIKey == acc2.APIKey {
		t.Error("should be different accounts")
	}
}

// --- Authenticate ---

func TestAuthenticatePlayerKey(t *testing.T) {
	pr := newTestRegistry(t, "")
	acc, _, _ := pr.FindOrCreateByDiscord("123", "Alice")

	name, admin, ok := pr.Authenticate(acc.APIKey)
	if !ok {
		t.Fatal("should succeed with valid player key")
	}
	if admin {
		t.Error("player key should not be admin")
	}
	if name != "Alice" {
		t.Errorf("expected Alice, got %s", name)
	}
}

func TestAuthenticateAdminKey(t *testing.T) {
	pr := newTestRegistry(t, "my-admin-key")

	name, admin, ok := pr.Authenticate("my-admin-key")
	if !ok {
		t.Fatal("should succeed with admin key")
	}
	if !admin {
		t.Error("should return admin=true")
	}
	if name != "" {
		t.Errorf("admin key should return empty player name, got %s", name)
	}
}

func TestAuthenticateInvalidKey(t *testing.T) {
	pr := newTestRegistry(t, "admin-key")
	pr.FindOrCreateByDiscord("123", "Alice")

	_, _, ok := pr.Authenticate("bogus-key")
	if ok {
		t.Error("should fail with invalid key")
	}
}

func TestAuthenticateEmptyKey(t *testing.T) {
	pr := newTestRegistry(t, "admin-key")
	_, _, ok := pr.Authenticate("")
	if ok {
		t.Error("should fail with empty key")
	}
}

// --- GetAccount ---

func TestGetAccount(t *testing.T) {
	pr := newTestRegistry(t, "")
	pr.FindOrCreateByDiscord("123", "Alice")

	acc := pr.GetAccount("alice")
	if acc == nil {
		t.Fatal("should find alice (case-insensitive)")
	}
	if acc.Name != "Alice" {
		t.Errorf("expected Alice, got %s", acc.Name)
	}

	if pr.GetAccount("nobody") != nil {
		t.Error("should return nil for unknown name")
	}
}

func TestGetAccountByDiscordID(t *testing.T) {
	pr := newTestRegistry(t, "")
	pr.FindOrCreateByDiscord("123", "Alice")

	acc := pr.GetAccountByDiscordID("123")
	if acc == nil {
		t.Fatal("should find by discord ID")
	}
	if acc.Name != "Alice" {
		t.Errorf("expected Alice, got %s", acc.Name)
	}

	if pr.GetAccountByDiscordID("999") != nil {
		t.Error("should return nil for unknown discord ID")
	}
}

// --- Persistence ---

func TestPersistenceSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	pr1 := &PlayerRegistry{
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		adminKey:   "admin",
		filePath:   fp,
	}
	acc1, _, _ := pr1.FindOrCreateByDiscord("111", "Alice")
	acc1.PlayerID = 1
	acc2, _, _ := pr1.FindOrCreateByDiscord("222", "Bob")
	acc2.PlayerID = 2
	pr1.mu.Lock()
	pr1.saveLocked()
	pr1.mu.Unlock()

	// Load into fresh registry
	pr2 := &PlayerRegistry{
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		adminKey:   "admin",
		filePath:   fp,
	}
	pr2.load()

	if len(pr2.discordIDs) != 2 {
		t.Fatalf("expected 2 accounts after load, got %d", len(pr2.discordIDs))
	}

	// API key auth works after load
	name, _, ok := pr2.Authenticate(acc1.APIKey)
	if !ok {
		t.Fatal("API key auth should work after load")
	}
	if name != "Alice" {
		t.Errorf("expected Alice, got %s", name)
	}

	// Discord ID lookup works after load
	loaded := pr2.GetAccountByDiscordID("222")
	if loaded == nil || loaded.Name != "Bob" {
		t.Error("discord ID lookup should work after load")
	}
	if loaded.PlayerID != 2 {
		t.Errorf("expected PlayerID 2, got %d", loaded.PlayerID)
	}
}

func TestPersistenceFileFormat(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	pr := &PlayerRegistry{
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		filePath:   fp,
	}
	pr.FindOrCreateByDiscord("123456", "Alice")

	data, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("failed to read accounts file: %v", err)
	}

	var saved []savedAccount
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("expected 1 saved account, got %d", len(saved))
	}
	if saved[0].DiscordID != "123456" {
		t.Errorf("expected discord ID 123456, got %s", saved[0].DiscordID)
	}
	if !strings.HasPrefix(saved[0].APIKey, "xk-") {
		t.Errorf("expected API key prefix xk-, got %s", saved[0].APIKey)
	}
}

func TestPersistenceFilePermissions(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	pr := &PlayerRegistry{
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		filePath:   fp,
	}
	pr.FindOrCreateByDiscord("123", "Alice")

	info, err := os.Stat(fp)
	if err != nil {
		t.Fatalf("failed to stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %04o", info.Mode().Perm())
	}
}

func TestPersistenceMissingFile(t *testing.T) {
	pr := &PlayerRegistry{
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		filePath:   filepath.Join(t.TempDir(), "nonexistent.json"),
	}
	pr.load()
	if len(pr.accounts) != 0 {
		t.Error("should have no accounts")
	}
}

func TestPersistenceSkipsLegacyAccounts(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	// Write a legacy account with no discord ID
	legacy := []savedAccount{
		{Name: "OldUser", DiscordID: "", APIKey: "xk-old", PlayerID: 1},
		{Name: "NewUser", DiscordID: "999", APIKey: "xk-new", PlayerID: 2},
	}
	data, _ := json.Marshal(legacy)
	os.WriteFile(fp, data, 0600)

	pr := &PlayerRegistry{
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		filePath:   fp,
	}
	pr.load()

	if len(pr.discordIDs) != 1 {
		t.Errorf("expected 1 account (legacy skipped), got %d", len(pr.discordIDs))
	}
	if pr.GetAccount("OldUser") != nil {
		t.Error("legacy account should be skipped")
	}
	if pr.GetAccount("NewUser") == nil {
		t.Error("discord account should be loaded")
	}
}

func TestPersistenceAutoSaveOnCreate(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "accounts.json")

	pr := &PlayerRegistry{
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		filePath:   fp,
	}
	pr.FindOrCreateByDiscord("123", "Alice")

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		t.Error("file should be auto-created")
	}
}

// --- Concurrency ---

func TestConcurrentOperations(t *testing.T) {
	pr := newTestRegistry(t, "admin")
	var wg sync.WaitGroup

	// Register 20 accounts concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := string(rune('A'+n)) + "discord"
			name := string(rune('A'+n)) + "user"
			pr.FindOrCreateByDiscord(id, name)
		}(i)
	}
	wg.Wait()

	pr.mu.RLock()
	count := len(pr.discordIDs)
	pr.mu.RUnlock()

	if count != 20 {
		t.Errorf("expected 20 accounts, got %d", count)
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
