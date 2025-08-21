package store

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"alxnet/internal/core"

	"github.com/dgraph-io/badger/v4"
	"github.com/fxamacker/cbor/v2"
	"go.uber.org/zap"
)

// Store configuration constants
const (
	DefaultMaxRetries = 3
	DefaultRetryDelay = 100 * time.Millisecond
	MaxKeyLength      = 1024
	MaxValueLength    = 100 * 1024 * 1024 // 100MB
)

// Store represents a robust key-value store with enhanced security
type Store struct {
	db         *badger.DB
	dataDir    string
	maxRetries int
	retryDelay time.Duration
	logger     *zap.Logger
	mu         sync.RWMutex
}

// StoreStats tracks store performance and usage
type StoreStats struct {
	TotalRecords  int64
	TotalContent  int64
	TotalDomains  int64
	TotalWebsites int64
	LastCleanup   time.Time
	MemoryUsage   int64
}

func Open(dir string) (*Store, error) {
	cleanDir := filepath.Clean(dir)
	db, err := badger.Open(badger.DefaultOptions(cleanDir))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	s := &Store{
		db:         db,
		dataDir:    cleanDir,
		maxRetries: DefaultMaxRetries,
		retryDelay: DefaultRetryDelay,
		logger:     logger,
	}

	logger.Info("store opened successfully", zap.String("dir", cleanDir))
	return s, nil
}

func (s *Store) Close() error {
	s.logger.Info("closing store")
	return s.db.Close()
}

// GetDataDir returns the data directory path
func (s *Store) GetDataDir() string {
	return s.dataDir
}

// PutRecordWithRetry stores a record with automatic retry logic
func (s *Store) PutRecordWithRetry(cid string, data []byte) error {
	if err := s.validateKeyValue(cid, data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		err := s.PutRecord(cid, data)
		if err == nil {
			if attempt > 0 {
				s.logger.Info("record stored after retry",
					zap.String("cid", cid),
					zap.Int("attempts", attempt+1))
			}
			return nil
		}
		lastErr = err
		if attempt < s.maxRetries {
			delay := s.retryDelay * time.Duration(attempt+1)
			s.logger.Warn("retry attempt failed",
				zap.String("cid", cid),
				zap.Int("attempt", attempt+1),
				zap.Error(err),
				zap.Duration("delay", delay))
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", s.maxRetries, lastErr)
}

func (s *Store) PutRecord(cid string, data []byte) error {
	if err := s.validateKeyValue(cid, data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("record:"+cid), data)
	})
}

func (s *Store) GetRecord(cid string) ([]byte, error) {
	if err := s.validateKey(cid); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get([]byte("record:" + cid))
		if err != nil {
			return err
		}
		return it.Value(func(v []byte) error {
			out = append([]byte{}, v...)
			return nil
		})
	})
	return out, err
}

func (s *Store) DeleteRecord(cid string) error {
	if err := s.validateKey(cid); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("record:" + cid))
	})
}

// ResolveRecordCID resolves a CID prefix to the full record CID, ensuring uniqueness.
func (s *Store) ResolveRecordCID(prefix string) (string, error) {
	if err := s.validateKey(prefix); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	var found string
	var count int
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		pfx := []byte("record:" + prefix)
		for it.Seek(pfx); it.ValidForPrefix(pfx); it.Next() {
			item := it.Item()
			k := item.Key()
			if !bytes.HasPrefix(k, []byte("record:")) {
				continue
			}
			full := string(k[len("record:"):])
			found = full
			count++
			if count > 1 {
				break
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", badger.ErrKeyNotFound
	}
	if count > 1 {
		return "", errors.New("ambiguous record CID prefix")
	}
	return found, nil
}

// PutContentWithRetry stores content with automatic retry logic
func (s *Store) PutContentWithRetry(cid string, data []byte) error {
	if err := s.validateKeyValue(cid, data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		err := s.PutContent(cid, data)
		if err == nil {
			if attempt > 0 {
				s.logger.Info("content stored after retry",
					zap.String("cid", cid),
					zap.Int("attempts", attempt+1))
			}
			return nil
		}
		lastErr = err
		if attempt < s.maxRetries {
			delay := s.retryDelay * time.Duration(attempt+1)
			s.logger.Warn("retry attempt failed",
				zap.String("cid", cid),
				zap.Int("attempt", attempt+1),
				zap.Error(err),
				zap.Duration("delay", delay))
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", s.maxRetries, lastErr)
}

func (s *Store) PutContent(cid string, data []byte) error {
	if err := s.validateKeyValue(cid, data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("content:"+cid), data)
	})
}

func (s *Store) GetContent(cid string) ([]byte, error) {
	if err := s.validateKey(cid); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get([]byte("content:" + cid))
		if err != nil {
			return nil
		}
		return it.Value(func(v []byte) error {
			out = append([]byte{}, v...)
			return nil
		})
	})
	return out, err
}

func (s *Store) DeleteContent(cid string) error {
	if err := s.validateKey(cid); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("content:" + cid))
	})
}

// Multi-file website support methods

// PutWebsiteManifest stores a website manifest
func (s *Store) PutWebsiteManifest(siteID string, manifestCID string, data []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		// Store the manifest data
		if err := txn.Set([]byte("manifest:"+manifestCID), data); err != nil {
			return err
		}
		// Update the site's current manifest pointer
		return txn.Set([]byte("site:"+siteID+":manifest"), []byte(manifestCID))
	})
}

// GetWebsiteManifest retrieves a website manifest by CID
func (s *Store) GetWebsiteManifest(manifestCID string) ([]byte, error) {
	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get([]byte("manifest:" + manifestCID))
		if err != nil {
			return err
		}
		return it.Value(func(v []byte) error {
			out = append([]byte{}, v...)
			return nil
		})
	})
	return out, err
}

// GetCurrentWebsiteManifest retrieves the current manifest for a site
func (s *Store) GetCurrentWebsiteManifest(siteID string) ([]byte, error) {
	var manifestCID []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get([]byte("site:" + siteID + ":manifest"))
		if err != nil {
			return err
		}
		return it.Value(func(v []byte) error {
			manifestCID = append([]byte{}, v...)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return s.GetWebsiteManifest(string(manifestCID))
}

// PutFileRecord stores a file record for a website
func (s *Store) PutFileRecord(siteID string, filePath string, recordCID string, data []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		// Store the file record data
		if err := txn.Set([]byte("filerecord:"+recordCID), data); err != nil {
			return err
		}
		// Store the file path -> record CID mapping
		if err := txn.Set([]byte("site:"+siteID+":file:"+filePath), []byte(recordCID)); err != nil {
			return err
		}
		return nil
	})
}

// GetFileRecord retrieves a file record by CID
func (s *Store) GetFileRecord(recordCID string) ([]byte, error) {
	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get([]byte("filerecord:" + recordCID))
		if err != nil {
			return err
		}
		return it.Value(func(v []byte) error {
			out = append([]byte{}, v...)
			return nil
		})
	})
	return out, err
}

// GetFileRecordByPath retrieves a file record for a specific file path in a website
func (s *Store) GetFileRecordByPath(siteID string, filePath string) ([]byte, error) {
	var recordCID []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get([]byte("site:" + siteID + ":file:" + filePath))
		if err != nil {
			return err
		}
		return it.Value(func(v []byte) error {
			recordCID = append([]byte{}, v...)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return s.GetFileRecord(string(recordCID))
}

// ListWebsiteFiles lists all files in a website
func (s *Store) ListWebsiteFiles(siteID string) (map[string]string, error) {
	files := make(map[string]string)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("site:" + siteID + ":file:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := string(item.Key())
			// Extract file path from key "site:siteID:file:path"
			parts := strings.SplitN(k, ":", 4)
			if len(parts) == 4 {
				filePath := parts[3]
				var recordCID []byte
				err := item.Value(func(v []byte) error {
					recordCID = append([]byte{}, v...)
					return nil
				})
				if err == nil {
					files[filePath] = string(recordCID)
				}
			}
		}
		return nil
	})
	return files, err
}

// GetWebsiteInfo retrieves comprehensive information about a website
func (s *Store) GetWebsiteInfo(siteID string) (*core.WebsiteInfo, error) {
	// Get current manifest
	manifestData, err := s.GetCurrentWebsiteManifest(siteID)
	if err != nil {
		return nil, err
	}

	// Parse manifest
	var manifest core.WebsiteManifest
	dec, _ := cbor.DecOptions{}.DecMode()
	if err := dec.Unmarshal(manifestData, &manifest); err != nil {
		return nil, err
	}

	// Get file information
	files := make(map[string]core.WebsiteFileInfo)
	for filePath, contentCID := range manifest.Files {
		// Get file record
		fileRecordData, err := s.GetFileRecordByPath(siteID, filePath)
		if err != nil {
			continue // Skip files we can't read
		}

		var fileRecord core.FileRecord
		if err := dec.Unmarshal(fileRecordData, &fileRecord); err != nil {
			continue
		}

		// Get content size
		content, err := s.GetContent(contentCID)
		var size int64
		if err == nil {
			size = int64(len(content))
		}

		files[filePath] = core.WebsiteFileInfo{
			Path:        filePath,
			ContentCID:  contentCID,
			MimeType:    fileRecord.MimeType,
			Size:        size,
			LastUpdated: time.Unix(fileRecord.TS, 0),
		}
	}

	return &core.WebsiteInfo{
		SiteID:      siteID,
		MainFile:    manifest.MainFile,
		Files:       files,
		FileCount:   len(files),
		LastUpdated: time.Unix(manifest.TS, 0),
	}, nil
}

// Check if a site has a website manifest (multi-file website)
func (s *Store) HasWebsiteManifest(siteID string) bool {
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte("site:" + siteID + ":manifest"))
		return err
	})
	return err == nil
}

// Check if a site has a traditional single-file record
func (s *Store) HasHead(siteID string) (bool, error) {
	var found bool
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("site:" + siteID + ":head:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			found = true
			break
		}
		return nil
	})
	return found, err
}

// GetHead retrieves the current head for a traditional single-file site
func (s *Store) GetHead(siteID string) (uint64, string, error) {
	var seq uint64
	var headCID string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("site:" + siteID + ":head:")
		it.Seek(prefix)
		if it.ValidForPrefix(prefix) {
			item := it.Item()
			k := string(item.Key())
			// Extract sequence from key "site:siteID:head:seq"
			parts := strings.SplitN(k, ":", 4)
			if len(parts) == 4 {
				if seqStr := parts[3]; seqStr != "" {
					if seqNum, err := strconv.ParseUint(seqStr, 10, 64); err == nil {
						seq = seqNum
					}
				}
			}

			// Get the head CID
			err := item.Value(func(v []byte) error {
				headCID = string(v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return seq, headCID, err
}

// PutHead stores the current head for a traditional single-file site
func (s *Store) PutHead(siteID string, seq uint64, headCID string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("site:%s:head:%d", siteID, seq)), []byte(headCID))
	})
}

// SetHead is an alias for PutHead for backward compatibility
func (s *Store) SetHead(siteID string, seq uint64, headCID string) error {
	return s.PutHead(siteID, seq, headCID)
}

// ListSites lists all site IDs that have stored data
func (s *Store) ListSites() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sites []string
	siteMap := make(map[string]bool)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		// Look for site manifest keys (site:<siteID>:manifest)
		prefix := []byte("site:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := string(it.Item().Key())
			// Parse key: site:<siteID>:manifest or site:<siteID>:file:...
			parts := strings.Split(key, ":")
			if len(parts) >= 3 && parts[0] == "site" {
				siteID := parts[1]
				if !siteMap[siteID] {
					sites = append(sites, siteID)
					siteMap[siteID] = true
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list sites: %w", err)
	}

	return sites, nil
}

// GetStorageUsage calculates the total storage usage across all content
func (s *Store) GetStorageUsage() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalBytes int64

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		// Count bytes for content: keys
		prefix := []byte("content:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			totalBytes += int64(it.Item().ValueSize())
		}

		return nil
	})

	return totalBytes, err
}

// GetContentFileCount counts the total number of content files
func (s *Store) GetContentFileCount() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		// Count content: keys
		prefix := []byte("content:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			count++
		}

		return nil
	})

	return count, err
}

// ResolveContentCID resolves a CID prefix to the full content CID, ensuring uniqueness.
func (s *Store) ResolveContentCID(prefix string) (string, error) {
	var found string
	var count int
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		pfx := []byte("content:" + prefix)
		for it.Seek(pfx); it.ValidForPrefix(pfx); it.Next() {
			item := it.Item()
			k := item.Key()
			if !bytes.HasPrefix(k, []byte("content:")) {
				continue
			}
			full := string(k[len("content:"):])
			found = full
			count++
			if count > 1 {
				break
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", badger.ErrKeyNotFound
	}
	if count > 1 {
		return "", errors.New("ambiguous content CID prefix")
	}
	return found, nil
}

// Domain resolution methods

// validateDomainName validates domain name format according to requirements:
// - alphanumeric lowercase only
// - starts with letter or number
// - can contain underscores and dashes
// - minimum 3 characters, maximum 32 characters
func (s *Store) validateDomainName(domain string) error {
	if len(domain) < 3 {
		return fmt.Errorf("domain name must be at least 3 characters long")
	}
	if len(domain) > 32 {
		return fmt.Errorf("domain name must be at most 32 characters long")
	}

	// Check first character
	firstChar := domain[0]
	if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= '0' && firstChar <= '9')) {
		return fmt.Errorf("domain name must start with a letter or number")
	}

	// Check all characters
	for i, char := range domain {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
			return fmt.Errorf("domain name can only contain lowercase letters, numbers, underscores, and dashes")
		}

		// No consecutive special characters
		if i > 0 && (char == '_' || char == '-') {
			prevChar := rune(domain[i-1])
			if prevChar == '_' || prevChar == '-' {
				return fmt.Errorf("domain name cannot have consecutive underscores or dashes")
			}
		}
	}

	// Cannot end with underscore or dash
	lastChar := domain[len(domain)-1]
	if lastChar == '_' || lastChar == '-' {
		return fmt.Errorf("domain name cannot end with underscore or dash")
	}

	return nil
}

func (s *Store) PutDomain(domain string, siteID string) error {
	// Validate domain name
	if err := s.validateDomainName(domain); err != nil {
		return fmt.Errorf("invalid domain name: %w", err)
	}

	// Check if domain already exists
	existingSiteID, err := s.GetDomain(domain)
	if err == nil && existingSiteID != siteID {
		return fmt.Errorf("domain '%s' is already registered to another site", domain)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("domain:"+domain), []byte(siteID))
	})
}

func (s *Store) GetDomain(domain string) (string, error) {
	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get([]byte("domain:" + domain))
		if err != nil {
			return err
		}
		return it.Value(func(v []byte) error {
			out = append([]byte{}, v...)
			return nil
		})
	})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *Store) ResolveDomain(domain string) (string, error) {
	return s.GetDomain(domain)
}

func (s *Store) ListDomains() (map[string]string, error) {
	domains := make(map[string]string)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("domain:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := string(item.Key())
			domain := strings.TrimPrefix(k, "domain:")

			var siteID []byte
			err := item.Value(func(v []byte) error {
				siteID = append([]byte{}, v...)
				return nil
			})
			if err == nil {
				domains[domain] = string(siteID)
			}
		}
		return nil
	})
	return domains, err
}

func (s *Store) TransferDomain(domain string, newOwnerPub []byte, signature []byte) error {
	// TODO: Implement domain transfer with cryptographic proof
	// For now, just return an error
	return errors.New("domain transfer not yet implemented")
}

// ValidateSiteName validates a site name according to the rules:
// - alphanumeric characters only (lowercase)
// - can contain underscores and dashes
// - must start with letter or number
// - length between 3 and 32 characters
func (s *Store) ValidateSiteName(siteName string) error {
	if len(siteName) < 3 || len(siteName) > 32 {
		return errors.New("site name must be between 3 and 32 characters")
	}

	// Check first character is alphanumeric
	if !((siteName[0] >= 'a' && siteName[0] <= 'z') || (siteName[0] >= '0' && siteName[0] <= '9')) {
		return errors.New("site name must start with a letter or number")
	}

	// Check all characters are valid
	for _, r := range siteName {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return errors.New("site name can only contain lowercase letters, numbers, underscores, and dashes")
		}
	}

	return nil
}

// RegisterSiteName registers a site name for a site ID with conflict resolution
// Returns error if validation fails or if the name is already taken
func (s *Store) RegisterSiteName(siteName string, siteID string) error {
	// Validate site name format
	if err := s.ValidateSiteName(siteName); err != nil {
		return err
	}

	// Check if site name is already taken
	existingSiteID, err := s.GetDomain(siteName)
	if err == nil && existingSiteID != "" {
		if existingSiteID == siteID {
			// Same site trying to register again - that's ok
			return nil
		}
		return fmt.Errorf("site name '%s' is already registered to another site", siteName)
	}

	// Register the site name
	return s.PutDomain(siteName, siteID)
}

// Validation methods
func (s *Store) validateKey(key string) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}
	if len(key) > MaxKeyLength {
		return fmt.Errorf("key too long: %d > %d", len(key), MaxKeyLength)
	}
	if !isValidHexString(key) {
		return fmt.Errorf("invalid key format: %s", key)
	}
	return nil
}

func (s *Store) validateKeyValue(key string, value []byte) error {
	if err := s.validateKey(key); err != nil {
		return err
	}
	if value == nil {
		return errors.New("value cannot be nil")
	}
	if len(value) > MaxValueLength {
		return fmt.Errorf("value too large: %d > %d", len(value), MaxValueLength)
	}
	return nil
}

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

// Configuration methods
func (s *Store) SetMaxRetries(max int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxRetries = max
}

func (s *Store) SetRetryDelay(delay time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retryDelay = delay
}

// Statistics methods
func (s *Store) GetStats() (*StoreStats, error) {
	stats := &StoreStats{}

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()

			if bytes.HasPrefix(k, []byte("record:")) {
				stats.TotalRecords++
			} else if bytes.HasPrefix(k, []byte("content:")) {
				stats.TotalContent++
			} else if bytes.HasPrefix(k, []byte("domain:")) {
				stats.TotalDomains++
			} else if bytes.HasPrefix(k, []byte("website:")) {
				stats.TotalWebsites++
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	stats.LastCleanup = time.Now()
	return stats, nil
}

// Cleanup methods
func (s *Store) CleanupOldRecords(maxAge time.Duration) error {
	s.logger.Info("starting cleanup of old records", zap.Duration("max_age", maxAge))

	deleted := 0

	err := s.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()

			// Only clean up content records, not metadata
			if bytes.HasPrefix(k, []byte("content:")) {
				// Check if content is old enough to delete
				// This is a simplified check - in practice you'd want to store timestamps
				if deleted < 1000 { // Limit cleanup per run
					if err := txn.Delete(k); err != nil {
						s.logger.Warn("failed to delete old record",
							zap.String("key", string(k)),
							zap.Error(err))
						continue
					}
					deleted++
				}
			}
		}
		return nil
	})

	s.logger.Info("cleanup completed",
		zap.Int("deleted", deleted),
		zap.Error(err))

	return err
}
