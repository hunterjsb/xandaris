package game

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PlayerAccount stores credentials for a registered player.
type PlayerAccount struct {
	Name      string `json:"name"`
	DiscordID string `json:"discord_id"`
	APIKey    string `json:"api_key"`
	PlayerID  int    `json:"player_id"`
}

// savedAccount is the on-disk representation.
type savedAccount struct {
	Name      string `json:"name"`
	DiscordID string `json:"discord_id"`
	APIKey    string `json:"api_key"`
	PlayerID  int    `json:"player_id"`
}

// PlayerRegistry manages player accounts and API key authentication.
type PlayerRegistry struct {
	mu         sync.RWMutex
	accounts   map[string]*PlayerAccount // lowercase name → account
	keys       map[string]*PlayerAccount // api_key → account
	discordIDs map[string]*PlayerAccount // discord_id → account
	adminKey   string                    // global admin key (XANDARIS_API_KEY)
	filePath   string                    // path to accounts.json
}

// NewPlayerRegistry creates a new registry with an optional admin key.
// It loads any existing accounts from ~/.xandaris/accounts.json.
func NewPlayerRegistry(adminKey string) *PlayerRegistry {
	home, _ := os.UserHomeDir()
	fp := filepath.Join(home, ".xandaris", "accounts.json")

	pr := &PlayerRegistry{
		accounts:   make(map[string]*PlayerAccount),
		keys:       make(map[string]*PlayerAccount),
		discordIDs: make(map[string]*PlayerAccount),
		adminKey:   adminKey,
		filePath:   fp,
	}
	pr.load()
	return pr
}

// FindOrCreateByDiscord looks up an account by Discord ID, or creates one.
// Returns the account and whether it was newly created.
func (pr *PlayerRegistry) FindOrCreateByDiscord(discordID, discordUsername string) (*PlayerAccount, bool, error) {
	if discordID == "" {
		return nil, false, fmt.Errorf("discord ID required")
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Existing account — update username if it changed
	if acc, exists := pr.discordIDs[discordID]; exists {
		if acc.Name != discordUsername {
			delete(pr.accounts, strings.ToLower(acc.Name))
			acc.Name = discordUsername
			pr.accounts[strings.ToLower(discordUsername)] = acc
			pr.saveLocked()
		}
		return acc, false, nil
	}

	// New account
	name := strings.TrimSpace(discordUsername)
	if len(name) > 24 {
		name = name[:24]
	}
	if len(name) < 1 {
		name = "Player"
	}

	apiKey := generateAPIKey()
	account := &PlayerAccount{
		Name:      name,
		DiscordID: discordID,
		APIKey:    apiKey,
	}

	pr.accounts[strings.ToLower(name)] = account
	pr.keys[apiKey] = account
	pr.discordIDs[discordID] = account

	pr.saveLocked()

	return account, true, nil
}

// FindOrCreateByName looks up an account by name, or creates one (for admin registration).
func (pr *PlayerRegistry) FindOrCreateByName(name string) (*PlayerAccount, bool, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 24 {
		return nil, false, fmt.Errorf("name must be 1-24 characters")
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	if acc, exists := pr.accounts[strings.ToLower(name)]; exists {
		return acc, false, nil
	}

	apiKey := generateAPIKey()
	account := &PlayerAccount{
		Name:   name,
		APIKey: apiKey,
	}
	pr.accounts[strings.ToLower(name)] = account
	pr.keys[apiKey] = account
	pr.saveLocked()

	return account, true, nil
}

// Authenticate checks an API key and returns the player name.
// Returns ("", true) for admin key, (playerName, false) for player key.
func (pr *PlayerRegistry) Authenticate(key string) (playerName string, isAdmin bool, ok bool) {
	if key == "" {
		return "", false, false
	}

	// Check admin key first
	if pr.adminKey != "" && key == pr.adminKey {
		return "", true, true
	}

	pr.mu.RLock()
	defer pr.mu.RUnlock()

	if account, exists := pr.keys[key]; exists {
		return account.Name, false, true
	}

	return "", false, false
}

// RemoveAccount deletes an account by name and persists.
func (pr *PlayerRegistry) RemoveAccount(name string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	key := strings.ToLower(name)
	acc, exists := pr.accounts[key]
	if !exists {
		return
	}
	delete(pr.accounts, key)
	if acc.APIKey != "" {
		delete(pr.keys, acc.APIKey)
	}
	if acc.DiscordID != "" {
		delete(pr.discordIDs, acc.DiscordID)
	}
	pr.saveLocked()
}

// GetAccount returns an account by name.
func (pr *PlayerRegistry) GetAccount(name string) *PlayerAccount {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.accounts[strings.ToLower(name)]
}

// GetAccountByDiscordID returns an account by Discord ID.
func (pr *PlayerRegistry) GetAccountByDiscordID(discordID string) *PlayerAccount {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.discordIDs[discordID]
}

// GetAllAccounts returns all registered accounts.
func (pr *PlayerRegistry) GetAllAccounts() []*PlayerAccount {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	result := make([]*PlayerAccount, 0, len(pr.accounts))
	for _, acc := range pr.accounts {
		result = append(result, acc)
	}
	return result
}

// Save persists all accounts to disk (thread-safe).
func (pr *PlayerRegistry) Save() {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	pr.saveLocked()
}

// saveLocked persists all accounts to disk. Caller must hold the lock.
func (pr *PlayerRegistry) saveLocked() {
	if pr.filePath == "" {
		return
	}
	// Save ALL accounts, not just Discord-linked ones
	seen := make(map[string]bool)
	var saved []savedAccount
	for _, acc := range pr.accounts {
		key := strings.ToLower(acc.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		saved = append(saved, savedAccount{
			Name:      acc.Name,
			DiscordID: acc.DiscordID,
			APIKey:    acc.APIKey,
			PlayerID:  acc.PlayerID,
		})
	}
	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		fmt.Printf("[Auth] Failed to marshal accounts: %v\n", err)
		return
	}
	os.MkdirAll(filepath.Dir(pr.filePath), 0700)
	if err := os.WriteFile(pr.filePath, data, 0600); err != nil {
		fmt.Printf("[Auth] Failed to save accounts: %v\n", err)
	}
}

// load reads accounts from disk. Called once at startup.
func (pr *PlayerRegistry) load() {
	if pr.filePath == "" {
		return
	}
	data, err := os.ReadFile(pr.filePath)
	if err != nil {
		return // file doesn't exist yet
	}
	var saved []savedAccount
	if err := json.Unmarshal(data, &saved); err != nil {
		fmt.Printf("[Auth] Failed to parse accounts file: %v\n", err)
		return
	}
	for _, s := range saved {
		acc := &PlayerAccount{
			Name:      s.Name,
			DiscordID: s.DiscordID,
			APIKey:    s.APIKey,
			PlayerID:  s.PlayerID,
		}
		pr.accounts[strings.ToLower(s.Name)] = acc
		if s.APIKey != "" {
			pr.keys[s.APIKey] = acc
		}
		if s.DiscordID != "" {
			pr.discordIDs[s.DiscordID] = acc
		}
	}
	fmt.Printf("[Auth] Loaded %d accounts from %s\n", len(pr.accounts), pr.filePath)
}

func generateAPIKey() string {
	bytes := make([]byte, 24)
	rand.Read(bytes)
	return "xk-" + hex.EncodeToString(bytes)
}
