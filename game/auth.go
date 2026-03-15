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

	"golang.org/x/crypto/bcrypt"
)

// PlayerAccount stores credentials for a registered player.
type PlayerAccount struct {
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	APIKey       string `json:"api_key"`
	PlayerID     int    `json:"player_id"`
}

// savedAccount is the on-disk representation (includes password hash).
type savedAccount struct {
	Name         string `json:"name"`
	PasswordHash string `json:"password_hash"`
	APIKey       string `json:"api_key"`
	PlayerID     int    `json:"player_id"`
}

// PlayerRegistry manages player accounts and API key authentication.
type PlayerRegistry struct {
	mu       sync.RWMutex
	accounts map[string]*PlayerAccount // name → account
	keys     map[string]*PlayerAccount // api_key → account
	adminKey string                    // global admin key (XANDARIS_API_KEY)
	filePath string                    // path to accounts.json
}

// NewPlayerRegistry creates a new registry with an optional admin key.
// It loads any existing accounts from ~/.xandaris/accounts.json.
func NewPlayerRegistry(adminKey string) *PlayerRegistry {
	home, _ := os.UserHomeDir()
	fp := filepath.Join(home, ".xandaris", "accounts.json")

	pr := &PlayerRegistry{
		accounts: make(map[string]*PlayerAccount),
		keys:     make(map[string]*PlayerAccount),
		adminKey: adminKey,
		filePath: fp,
	}
	pr.load()
	return pr
}

// Register creates a new player account. Returns the account or error.
func (pr *PlayerRegistry) Register(name, password string) (*PlayerAccount, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 24 {
		return nil, fmt.Errorf("name must be 2-24 characters")
	}
	if len(password) < 4 {
		return nil, fmt.Errorf("password must be at least 4 characters")
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.accounts[strings.ToLower(name)]; exists {
		return nil, fmt.Errorf("name already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password")
	}

	apiKey := generateAPIKey()

	account := &PlayerAccount{
		Name:         name,
		PasswordHash: string(hash),
		APIKey:       apiKey,
	}

	pr.accounts[strings.ToLower(name)] = account
	pr.keys[apiKey] = account

	pr.saveLocked()

	return account, nil
}

// Login verifies credentials and returns the account.
func (pr *PlayerRegistry) Login(name, password string) (*PlayerAccount, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	account, exists := pr.accounts[strings.ToLower(name)]
	if !exists {
		return nil, fmt.Errorf("unknown player")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("wrong password")
	}

	return account, nil
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

// GetAccount returns an account by name.
func (pr *PlayerRegistry) GetAccount(name string) *PlayerAccount {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.accounts[strings.ToLower(name)]
}

// Save persists all accounts to disk. Caller must hold at least a read lock.
func (pr *PlayerRegistry) saveLocked() {
	if pr.filePath == "" {
		return
	}
	var saved []savedAccount
	for _, acc := range pr.accounts {
		saved = append(saved, savedAccount{
			Name:         acc.Name,
			PasswordHash: acc.PasswordHash,
			APIKey:       acc.APIKey,
			PlayerID:     acc.PlayerID,
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
		return // file doesn't exist yet — that's fine
	}
	var saved []savedAccount
	if err := json.Unmarshal(data, &saved); err != nil {
		fmt.Printf("[Auth] Failed to parse accounts file: %v\n", err)
		return
	}
	for _, s := range saved {
		acc := &PlayerAccount{
			Name:         s.Name,
			PasswordHash: s.PasswordHash,
			APIKey:       s.APIKey,
			PlayerID:     s.PlayerID,
		}
		pr.accounts[strings.ToLower(s.Name)] = acc
		pr.keys[s.APIKey] = acc
	}
	fmt.Printf("[Auth] Loaded %d accounts from %s\n", len(saved), pr.filePath)
}

func generateAPIKey() string {
	bytes := make([]byte, 24)
	rand.Read(bytes)
	return "xk-" + hex.EncodeToString(bytes)
}
