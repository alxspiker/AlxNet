package wallet

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"betanet/internal/core"

	"github.com/tyler-smith/go-bip39"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	walletVersion = 1
	kdfName       = "argon2id"
	adWallet      = "bn-wallet-v1"
	adContent     = "bn-content-v1"
	contentHdr    = "BNE1" // 4 bytes

	// Security constants
	MinMnemonicLength   = 12
	MaxMnemonicLength   = 24
	MaxLabelLength      = 100
	MaxPassphraseLength = 256
	MinPassphraseLength = 8
	MaxWalletSize       = 10 * 1024 * 1024 // 10MB
	MaxSitesPerWallet   = 1000
)

// Security configuration
type SecurityConfig struct {
	EnableRateLimiting      bool
	MaxAttemptsPerMinute    int
	LockoutDuration         time.Duration
	RequireStrongPassphrase bool
}

// DefaultSecurityConfig returns sensible security defaults
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnableRateLimiting:      true,
		MaxAttemptsPerMinute:    5,
		LockoutDuration:         15 * time.Minute,
		RequireStrongPassphrase: true,
	}
}

type SiteMeta struct {
	Label       string    `json:"label"`
	SiteID      string    `json:"site_id"`
	SitePubHex  string    `json:"site_pub"`
	Seq         uint64    `json:"seq"`
	HeadRecCID  string    `json:"head_rec_cid"`
	ContentCID  string    `json:"content_cid"`
	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
}

// Validate performs comprehensive validation of SiteMeta
func (sm *SiteMeta) Validate() error {
	if sm.Label == "" {
		return errors.New("label is required")
	}
	if len(sm.Label) > MaxLabelLength {
		return fmt.Errorf("label too long: %d > %d", len(sm.Label), MaxLabelLength)
	}
	if sm.SiteID == "" {
		return errors.New("site ID is required")
	}
	if !isValidHexString(sm.SiteID) {
		return fmt.Errorf("invalid site ID format: %s", sm.SiteID)
	}
	if sm.SitePubHex == "" {
		return errors.New("site public key is required")
	}
	if !isValidHexString(sm.SitePubHex) {
		return fmt.Errorf("invalid site public key format: %s", sm.SitePubHex)
	}
	if sm.Seq == 0 {
		return errors.New("sequence number must be positive")
	}
	if sm.CreatedAt.IsZero() {
		return errors.New("creation time is required")
	}
	if sm.LastUpdated.IsZero() {
		return errors.New("last updated time is required")
	}
	if sm.LastUpdated.Before(sm.CreatedAt) {
		return errors.New("last updated cannot be before creation time")
	}
	return nil
}

type Wallet struct {
	Version        int                  `json:"v"`
	Sites          map[string]*SiteMeta `json:"sites"` // key = label
	CreatedAt      time.Time            `json:"created_at"`
	LastAccessed   time.Time            `json:"last_accessed"`
	SecurityConfig *SecurityConfig      `json:"security_config,omitempty"`
}

// Validate performs comprehensive validation of Wallet
func (w *Wallet) Validate() error {
	if w.Version != walletVersion {
		return fmt.Errorf("unsupported wallet version: %d", w.Version)
	}
	if w.Sites == nil {
		return errors.New("sites map cannot be nil")
	}
	if len(w.Sites) > MaxSitesPerWallet {
		return fmt.Errorf("too many sites: %d > %d", len(w.Sites), MaxSitesPerWallet)
	}
	if w.CreatedAt.IsZero() {
		return errors.New("creation time is required")
	}
	if w.LastAccessed.IsZero() {
		return errors.New("last accessed time is required")
	}

	// Validate each site
	for label, site := range w.Sites {
		if err := site.Validate(); err != nil {
			return fmt.Errorf("invalid site %s: %w", label, err)
		}
	}

	return nil
}

type encFile struct {
	Version   int    `json:"v"`
	KDF       string `json:"kdf"`
	SaltB64   string `json:"salt"`
	T         uint32 `json:"t"`
	MiB       uint32 `json:"m"`
	P         uint8  `json:"p"`
	NonceB64  string `json:"nonce"`
	CipherB64 string `json:"ct"`
}

// WalletManager handles wallet operations with security features
type WalletManager struct {
	config      *SecurityConfig
	logger      *zap.Logger
	rateLimiter map[string][]time.Time
	mu          sync.RWMutex //nolint:unused // For future thread-safe operations
}

// NewWalletManager creates a new wallet manager with security features
func NewWalletManager(config *SecurityConfig) (*WalletManager, error) {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &WalletManager{
		config:      config,
		logger:      logger,
		rateLimiter: make(map[string][]time.Time),
	}, nil
}

// ValidateMnemonic validates a mnemonic phrase for security
func ValidateMnemonic(mnemonic string) error {
	if mnemonic == "" {
		return errors.New("mnemonic cannot be empty")
	}

	words := strings.Fields(mnemonic)
	if len(words) < MinMnemonicLength || len(words) > MaxMnemonicLength {
		return fmt.Errorf("mnemonic must have between %d and %d words, got %d",
			MinMnemonicLength, MaxMnemonicLength, len(words))
	}

	// Check if mnemonic is valid according to BIP-39
	if !bip39.IsMnemonicValid(mnemonic) {
		return errors.New("invalid mnemonic phrase")
	}

	// Check for common weak patterns
	if isWeakMnemonic(mnemonic) {
		return errors.New("mnemonic is too weak - avoid common patterns")
	}

	return nil
}

// ValidatePassphrase validates a passphrase for security
func ValidatePassphrase(passphrase string, requireStrong bool) error {
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}

	if len(passphrase) < MinPassphraseLength {
		return fmt.Errorf("passphrase too short: %d < %d", len(passphrase), MinPassphraseLength)
	}

	if len(passphrase) > MaxPassphraseLength {
		return fmt.Errorf("passphrase too long: %d > %d", len(passphrase), MaxPassphraseLength)
	}

	if requireStrong {
		if err := validateStrongPassphrase(passphrase); err != nil {
			return fmt.Errorf("passphrase too weak: %w", err)
		}
	}

	return nil
}

// Rate limiting methods
func (wm *WalletManager) checkRateLimit(identifier string) error { //nolint:unused // For future rate limiting
	if !wm.config.EnableRateLimiting {
		return nil
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	now := time.Now()
	window := time.Minute

	if requests, exists := wm.rateLimiter[identifier]; exists {
		// Remove old requests outside window
		var valid []time.Time
		for _, req := range requests {
			if now.Sub(req) < window {
				valid = append(valid, req)
			}
		}
		wm.rateLimiter[identifier] = valid

		if len(valid) >= wm.config.MaxAttemptsPerMinute {
			return fmt.Errorf("rate limit exceeded: %d attempts per minute", wm.config.MaxAttemptsPerMinute)
		}
	}

	wm.rateLimiter[identifier] = append(wm.rateLimiter[identifier], now)
	return nil
}

func NewMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}

func masterKeyFromMnemonic(mnemonic string) ([]byte, error) {
	if err := ValidateMnemonic(mnemonic); err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %w", err)
	}

	seed := bip39.NewSeed(mnemonic, "")
	h := hkdf.New(sha256.New, seed, []byte("bn-wallet-v1"), []byte("master"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, err
	}
	return key, nil
}

func DeriveSiteKey(master []byte, label string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	if len(master) != 32 {
		return nil, nil, fmt.Errorf("invalid master key length: %d", len(master))
	}

	if label == "" {
		return nil, nil, errors.New("label cannot be empty")
	}

	if len(label) > MaxLabelLength {
		return nil, nil, fmt.Errorf("label too long: %d > %d", len(label), MaxLabelLength)
	}

	h := hkdf.New(sha256.New, master, []byte("bn-site"), []byte(strings.ToLower(label)))
	seed := make([]byte, 32)
	if _, err := io.ReadFull(h, seed); err != nil {
		return nil, nil, err
	}
	priv := ed25519.NewKeyFromSeed(seed)
	return priv.Public().(ed25519.PublicKey), priv, nil
}

func EncryptWallet(w *Wallet, mnemonic string) ([]byte, error) {
	if err := w.Validate(); err != nil {
		return nil, fmt.Errorf("invalid wallet: %w", err)
	}

	if err := ValidateMnemonic(mnemonic); err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %w", err)
	}

	raw, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	t, mMiB, p := uint32(2), uint32(64), uint8(4)
	key := argon2.IDKey([]byte(mnemonic), salt, t, mMiB*1024, uint8(p), 32)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ct := aead.Seal(nil, nonce, raw, []byte(adWallet))
	out := encFile{
		Version:   walletVersion,
		KDF:       kdfName,
		SaltB64:   base64.StdEncoding.EncodeToString(salt),
		T:         t,
		MiB:       mMiB,
		P:         p,
		NonceB64:  base64.StdEncoding.EncodeToString(nonce),
		CipherB64: base64.StdEncoding.EncodeToString(ct),
	}
	return json.Marshal(out)
}

func DecryptWallet(encBytes []byte, mnemonic string) (*Wallet, error) {
	var ef encFile
	if err := json.Unmarshal(encBytes, &ef); err != nil {
		return nil, err
	}
	if ef.KDF != kdfName || ef.Version != walletVersion {
		return nil, errors.New("unsupported wallet format")
	}
	salt, err := base64.StdEncoding.DecodeString(ef.SaltB64)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(ef.NonceB64)
	if err != nil {
		return nil, err
	}
	ct, err := base64.StdEncoding.DecodeString(ef.CipherB64)
	if err != nil {
		return nil, err
	}

	key := argon2.IDKey([]byte(mnemonic), salt, ef.T, ef.MiB*1024, uint8(ef.P), 32)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	raw, err := aead.Open(nil, nonce, ct, []byte(adWallet))
	if err != nil {
		return nil, errors.New("bad mnemonic or corrupted wallet")
	}
	var w Wallet
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func Save(path string, bytes []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0o600)
}

func Load(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func New() *Wallet {
	now := time.Now()
	return &Wallet{
		Version:      walletVersion,
		Sites:        map[string]*SiteMeta{},
		CreatedAt:    now,
		LastAccessed: now,
	}
}

func (w *Wallet) EnsureSite(master []byte, label string) (*SiteMeta, ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := DeriveSiteKey(master, label)
	if err != nil {
		return nil, nil, nil, err
	}
	siteID := core.SiteIDFromPub(pub)
	if m, ok := w.Sites[label]; ok {
		return m, pub, priv, nil
	}
	// hex encode pub
	const hexdigits = "0123456789abcdef"
	hexOf := func(b []byte) string {
		out := make([]byte, len(b)*2)
		for i, v := range b {
			out[i*2] = hexdigits[v>>4]
			out[i*2+1] = hexdigits[v&0x0f]
		}
		return string(out)
	}
	now := time.Now()
	meta := &SiteMeta{
		Label:       label,
		SiteID:      siteID,
		SitePubHex:  strings.ToLower(hexOf(pub)),
		Seq:         1, // Start with sequence number 1
		CreatedAt:   now,
		LastUpdated: now,
	}
	w.Sites[label] = meta
	return meta, pub, priv, nil
}

// Encrypt content with passphrase. Output: "BNE1" || salt(16) || nonce(24) || ciphertext
func EncryptContent(passphrase string, plaintext []byte) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key := argon2.IDKey([]byte(passphrase), salt, 2, 64*1024, 4, 32)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := aead.Seal(nil, nonce, plaintext, []byte(adContent))
	out := make([]byte, 0, 4+16+len(nonce)+len(ct))
	out = append(out, contentHdr...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func DecryptContent(passphrase string, blob []byte) ([]byte, error) {
	if len(blob) < 4+16+chacha20poly1305.NonceSizeX {
		return nil, errors.New("invalid blob")
	}
	if string(blob[:4]) != contentHdr {
		return nil, errors.New("bad header")
	}
	salt := blob[4:20]
	nonce := blob[20 : 20+chacha20poly1305.NonceSizeX]
	ct := blob[20+chacha20poly1305.NonceSizeX:]
	key := argon2.IDKey([]byte(passphrase), salt, 2, 64*1024, 4, 32)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, ct, []byte(adContent))
}

// Utility functions
func isValidHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	if len(s)%2 != 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func isWeakMnemonic(mnemonic string) bool {
	// Check for common weak patterns
	weakPatterns := []string{
		"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
		"zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo wrong",
		"test test test test test test test test test test test junk",
	}

	normalized := strings.ToLower(strings.TrimSpace(mnemonic))
	for _, pattern := range weakPatterns {
		if normalized == pattern {
			return true
		}
	}

	// Check for repeated words
	words := strings.Fields(normalized)
	wordCount := make(map[string]int)
	for _, word := range words {
		wordCount[word]++
		if wordCount[word] > 3 {
			return true // Too many repeated words
		}
	}

	return false
}

func validateStrongPassphrase(passphrase string) error {
	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
		length     = len(passphrase)
	)

	for _, char := range passphrase {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case char >= 33 && char <= 126: // Printable ASCII special characters
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("must contain uppercase letter")
	}
	if !hasLower {
		return errors.New("must contain lowercase letter")
	}
	if !hasNumber {
		return errors.New("must contain number")
	}
	if !hasSpecial {
		return errors.New("must contain special character")
	}
	if length < 12 {
		return errors.New("must be at least 12 characters")
	}

	return nil
}
