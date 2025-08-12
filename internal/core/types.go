package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// Security constants
const (
	MaxContentSize    = 10 * 1024 * 1024 // 10MB limit
	MaxFileCount      = 1000             // Maximum files per website
	MaxPathLength     = 255              // Maximum file path length
	MaxRecordSize     = 1024 * 1024      // 1MB limit for records
	MinSequenceNumber = 1
	MaxSequenceNumber = 1<<63 - 1 // Max uint63
)

// Allowed file extensions for security
var AllowedExtensions = []string{
	".html", ".htm", ".css", ".js", ".png", ".jpg", ".jpeg",
	".gif", ".svg", ".ico", ".json", ".xml", ".txt", ".md",
	".woff", ".woff2", ".ttf", ".eot", ".webp", ".avif",
}

// UpdateRecord is the canonical, signed statement that a site's content head
// moves to a new content CID. Authorization uses a master Site key that links
// a per-update ephemeral key via LinkSig; the update body is signed by the
// ephemeral key. The record is encoded with canonical CBOR.
type UpdateRecord struct {
	Version    string `cbor:"0,keyasint"`
	SitePub    []byte `cbor:"1,keyasint"` // 32B ed25519 pub
	Seq        uint64 `cbor:"2,keyasint"`
	PrevCID    string `cbor:"3,keyasint"` // hex sha256 of previous record encoding
	ContentCID string `cbor:"4,keyasint"` // hex sha256 of content bytes
	TS         int64  `cbor:"5,keyasint"` // unix seconds
	UpdatePub  []byte `cbor:"6,keyasint"` // 32B ed25519 pub (ephemeral per update)
	LinkSig    []byte `cbor:"7,keyasint"` // sig by SitePriv over link preimage
	UpdateSig  []byte `cbor:"8,keyasint"` // sig by UpdatePriv over record preimage
}

// Validate performs comprehensive validation of an UpdateRecord
func (ur *UpdateRecord) Validate() error {
	if ur.Version == "" {
		return errors.New("version is required")
	}
	if len(ur.SitePub) != 32 {
		return fmt.Errorf("invalid site public key length: %d (expected 32)", len(ur.SitePub))
	}
	if ur.Seq < MinSequenceNumber || ur.Seq > MaxSequenceNumber {
		return fmt.Errorf("sequence number out of range: %d (must be between %d and %d)", ur.Seq, MinSequenceNumber, MaxSequenceNumber)
	}
	if ur.TS <= 0 {
		return fmt.Errorf("invalid timestamp: %d", ur.TS)
	}
	if ur.TS > time.Now().Unix()+3600 { // Allow 1 hour clock skew
		return fmt.Errorf("timestamp too far in future: %d", ur.TS)
	}
	if len(ur.UpdatePub) != 32 {
		return fmt.Errorf("invalid update public key length: %d (expected 32)", len(ur.UpdatePub))
	}
	if len(ur.LinkSig) != 64 {
		return fmt.Errorf("invalid link signature length: %d (expected 64)", len(ur.LinkSig))
	}
	if len(ur.UpdateSig) != 64 {
		return fmt.Errorf("invalid update signature length: %d (expected 64)", len(ur.UpdateSig))
	}
	if ur.PrevCID == "" && ur.Seq > 1 {
		return errors.New("previous CID required for non-initial sequence")
	}
	if ur.ContentCID == "" {
		return errors.New("content CID is required")
	}
	if !isValidHexString(ur.PrevCID) && ur.PrevCID != "" {
		return fmt.Errorf("invalid previous CID format: %s", ur.PrevCID)
	}
	if !isValidHexString(ur.ContentCID) {
		return fmt.Errorf("invalid content CID format: %s", ur.ContentCID)
	}
	return nil
}

// WebsiteManifest represents a multi-file website with references to all its files
type WebsiteManifest struct {
	Version   string            `cbor:"0,keyasint"`
	SitePub   []byte            `cbor:"1,keyasint"` // 32B ed25519 pub
	Seq       uint64            `cbor:"2,keyasint"`
	PrevCID   string            `cbor:"3,keyasint"` // hex sha256 of previous manifest
	TS        int64             `cbor:"4,keyasint"` // unix seconds
	MainFile  string            `cbor:"5,keyasint"` // path to main entry point (e.g., "index.html")
	Files     map[string]string `cbor:"6,keyasint"` // path -> content CID mapping
	UpdatePub []byte            `cbor:"7,keyasint"` // 32B ed25519 pub (ephemeral per update)
	LinkSig   []byte            `cbor:"8,keyasint"` // sig by SitePriv over link preimage
	UpdateSig []byte            `cbor:"9,keyasint"` // sig by UpdatePriv over manifest preimage
}

// Validate performs comprehensive validation of a WebsiteManifest
func (wm *WebsiteManifest) Validate() error {
	if wm.Version == "" {
		return errors.New("version is required")
	}
	if len(wm.SitePub) != 32 {
		return fmt.Errorf("invalid site public key length: %d (expected 32)", len(wm.SitePub))
	}
	if wm.Seq < MinSequenceNumber || wm.Seq > MaxSequenceNumber {
		return fmt.Errorf("sequence number out of range: %d (must be between %d and %d)", wm.Seq, MinSequenceNumber, MaxSequenceNumber)
	}
	if wm.TS <= 0 {
		return fmt.Errorf("invalid timestamp: %d", wm.TS)
	}
	if wm.TS > time.Now().Unix()+3600 { // Allow 1 hour clock skew
		return fmt.Errorf("timestamp too far in future: %d", wm.TS)
	}
	if wm.MainFile == "" {
		return errors.New("main file is required")
	}
	if err := ValidateFilePath(wm.MainFile); err != nil {
		return fmt.Errorf("invalid main file path: %w", err)
	}
	if len(wm.Files) > MaxFileCount {
		return fmt.Errorf("too many files: %d (maximum %d)", len(wm.Files), MaxFileCount)
	}
	for path, cid := range wm.Files {
		if err := ValidateFilePath(path); err != nil {
			return fmt.Errorf("invalid file path %s: %w", path, err)
		}
		if !isValidHexString(cid) {
			return fmt.Errorf("invalid content CID for %s: %s", path, cid)
		}
	}
	if len(wm.UpdatePub) != 32 {
		return fmt.Errorf("invalid update public key length: %d (expected 32)", len(wm.UpdatePub))
	}
	if len(wm.LinkSig) != 64 {
		return fmt.Errorf("invalid link signature length: %d (expected 64)", len(wm.LinkSig))
	}
	if len(wm.UpdateSig) != 64 {
		return fmt.Errorf("invalid update signature length: %d (expected 64)", len(wm.UpdateSig))
	}
	return nil
}

// FileRecord represents a file within a multi-file website
type FileRecord struct {
	Version    string `cbor:"0,keyasint"`
	SitePub    []byte `cbor:"1,keyasint"` // 32B ed25519 pub
	Path       string `cbor:"2,keyasint"` // file path within website (e.g., "styles/main.css")
	ContentCID string `cbor:"3,keyasint"` // hex sha256 of file content
	MimeType   string `cbor:"4,keyasint"` // MIME type of the file
	TS         int64  `cbor:"5,keyasint"` // unix seconds
	UpdatePub  []byte `cbor:"6,keyasint"` // 32B ed25519 pub (ephemeral per update)
	LinkSig    []byte `cbor:"7,keyasint"` // sig by SitePriv over link preimage
	UpdateSig  []byte `cbor:"8,keyasint"` // sig by UpdatePriv over file record preimage
}

// Validate performs comprehensive validation of a FileRecord
func (fr *FileRecord) Validate() error {
	if fr.Version == "" {
		return errors.New("version is required")
	}
	if len(fr.SitePub) != 32 {
		return fmt.Errorf("invalid site public key length: %d (expected 32)", len(fr.SitePub))
	}
	if fr.Path == "" {
		return errors.New("file path is required")
	}
	if err := ValidateFilePath(fr.Path); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	if fr.ContentCID == "" {
		return errors.New("content CID is required")
	}
	if !isValidHexString(fr.ContentCID) {
		return fmt.Errorf("invalid content CID format: %s", fr.ContentCID)
	}
	if fr.MimeType == "" {
		return errors.New("MIME type is required")
	}
	if !isValidMimeType(fr.MimeType) {
		return fmt.Errorf("invalid MIME type: %s", fr.MimeType)
	}
	if fr.TS <= 0 {
		return fmt.Errorf("invalid timestamp: %d", fr.TS)
	}
	if fr.TS > time.Now().Unix()+3600 { // Allow 1 hour clock skew
		return fmt.Errorf("timestamp too far in future: %d", fr.TS)
	}
	if len(fr.UpdatePub) != 32 {
		return fmt.Errorf("invalid update public key length: %d (expected 32)", len(fr.UpdatePub))
	}
	if len(fr.LinkSig) != 64 {
		return fmt.Errorf("invalid link signature length: %d (expected 64)", len(fr.LinkSig))
	}
	if len(fr.UpdateSig) != 64 {
		return fmt.Errorf("invalid update signature length: %d (expected 64)", len(fr.UpdateSig))
	}
	return nil
}

// DeleteRecord authorizes deletion of a specific record/content for a site.
type DeleteRecord struct {
	Version    string `cbor:"0,keyasint"`
	SitePub    []byte `cbor:"1,keyasint"`
	TargetRec  string `cbor:"2,keyasint"` // record CID to delete (optional)
	TargetCont string `cbor:"3,keyasint"` // content CID to delete (optional)
	TS         int64  `cbor:"4,keyasint"`
	Sig        []byte `cbor:"5,keyasint"` // Ed25519 by SitePriv over PreimageDelete
}

// Validate performs comprehensive validation of a DeleteRecord
func (dr *DeleteRecord) Validate() error {
	if dr.Version == "" {
		return errors.New("version is required")
	}
	if len(dr.SitePub) != 32 {
		return fmt.Errorf("invalid site public key length: %d (expected 32)", len(dr.SitePub))
	}
	if dr.TargetRec == "" && dr.TargetCont == "" {
		return errors.New("at least one target (record or content) must be specified")
	}
	if dr.TargetRec != "" && !isValidHexString(dr.TargetRec) {
		return fmt.Errorf("invalid target record CID format: %s", dr.TargetRec)
	}
	if dr.TargetCont != "" && !isValidHexString(dr.TargetCont) {
		return fmt.Errorf("invalid target content CID format: %s", dr.TargetCont)
	}
	if dr.TS <= 0 {
		return fmt.Errorf("invalid timestamp: %d", dr.TS)
	}
	if dr.TS > time.Now().Unix()+3600 { // Allow 1 hour clock skew
		return fmt.Errorf("timestamp too far in future: %d", dr.TS)
	}
	if len(dr.Sig) != 64 {
		return fmt.Errorf("invalid signature length: %d (expected 64)", len(dr.Sig))
	}
	return nil
}

// WebsiteFileInfo provides metadata about a file in a website
type WebsiteFileInfo struct {
	Path        string    `json:"path"`
	ContentCID  string    `json:"content_cid"`
	MimeType    string    `json:"mime_type"`
	Size        int64     `json:"size"`
	LastUpdated time.Time `json:"last_updated"`
}

// WebsiteInfo provides metadata about a complete website
type WebsiteInfo struct {
	SiteID      string                     `json:"site_id"`
	MainFile    string                     `json:"main_file"`
	Files       map[string]WebsiteFileInfo `json:"files"`
	FileCount   int                        `json:"file_count"`
	LastUpdated time.Time                  `json:"last_updated"`
}

func CanonicalMarshalNoUpdateSig(r *UpdateRecord) ([]byte, error) {
	tmp := *r
	tmp.UpdateSig = nil
	enc, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(tmp)
}

func CanonicalMarshal(r *UpdateRecord) ([]byte, error) {
	enc, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(r)
}

func CanonicalMarshalWebsiteManifest(wm *WebsiteManifest) ([]byte, error) {
	enc, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(wm)
}

func CanonicalMarshalWebsiteManifestNoUpdateSig(wm *WebsiteManifest) ([]byte, error) {
	tmp := *wm
	tmp.UpdateSig = nil
	enc, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(tmp)
}

func CanonicalMarshalFileRecord(fr *FileRecord) ([]byte, error) {
	enc, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(fr)
}

func CanonicalMarshalFileRecordNoUpdateSig(fr *FileRecord) ([]byte, error) {
	tmp := *fr
	tmp.UpdateSig = nil
	enc, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(tmp)
}

func CIDForBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func CIDForContent(content []byte) string {
	return CIDForBytes(content)
}

func NowTS() int64 {
	return time.Now().Unix()
}

func SiteIDFromPub(sitePub []byte) string {
	sum := sha256.Sum256(sitePub)
	return hex.EncodeToString(sum[:])
}

// IsHTMLFile checks if a file path represents an HTML file
func IsHTMLFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".html") ||
		strings.HasSuffix(strings.ToLower(path), ".htm")
}

// IsCSSFile checks if a file path represents a CSS file
func IsCSSFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".css")
}

// IsJSFile checks if a file path represents a JavaScript file
func IsJSFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".js")
}

// IsImageFile checks if a file path represents an image file
func IsImageFile(path string) bool {
	ext := strings.ToLower(path)
	return strings.HasSuffix(ext, ".png") ||
		strings.HasSuffix(ext, ".jpg") ||
		strings.HasSuffix(ext, ".jpeg") ||
		strings.HasSuffix(ext, ".gif") ||
		strings.HasSuffix(ext, ".svg") ||
		strings.HasSuffix(ext, ".ico")
}

// GetMimeType returns the MIME type for a file based on its extension
func GetMimeType(path string) string {
	ext := strings.ToLower(path)
	switch {
	case IsHTMLFile(path):
		return "text/html"
	case IsCSSFile(path):
		return "text/css"
	case IsJSFile(path):
		return "application/javascript"
	case strings.HasSuffix(ext, ".png"):
		return "image/png"
	case strings.HasSuffix(ext, ".jpg"), strings.HasSuffix(ext, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(ext, ".gif"):
		return "image/gif"
	case strings.HasSuffix(ext, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(ext, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(ext, ".json"):
		return "application/json"
	case strings.HasSuffix(ext, ".xml"):
		return "application/xml"
	case strings.HasSuffix(ext, ".txt"):
		return "text/plain"
	case strings.HasSuffix(ext, ".md"):
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// Utility functions for validation
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

func isValidMimeType(mimeType string) bool {
	// Basic MIME type validation
	if mimeType == "" {
		return false
	}

	// Check for common valid MIME types
	validTypes := []string{
		"text/html", "text/css", "application/javascript", "application/json",
		"image/png", "image/jpeg", "image/gif", "image/svg+xml", "image/x-icon",
		"text/plain", "text/markdown", "application/xml",
		"font/woff", "font/woff2", "font/ttf", "font/eot",
		"image/webp", "image/avif",
	}

	for _, valid := range validTypes {
		if mimeType == valid {
			return true
		}
	}

	// Allow custom MIME types with proper format
	if strings.Contains(mimeType, "/") && len(mimeType) <= 127 {
		return true
	}

	return false
}

// ValidateFilePath validates a file path for security and correctness
func ValidateFilePath(path string) error {
	if path == "" {
		return errors.New("file path cannot be empty")
	}

	if len(path) > MaxPathLength {
		return fmt.Errorf("file path too long: %d > %d", len(path), MaxPathLength)
	}

	// Prevent path traversal attacks
	if strings.Contains(path, "..") || strings.Contains(path, "//") {
		return errors.New("invalid file path: contains path traversal elements")
	}

	// Check for absolute paths
	if filepath.IsAbs(path) {
		return errors.New("absolute file paths are not allowed")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return errors.New("file must have a valid extension")
	}

	allowed := false
	for _, allowedExt := range AllowedExtensions {
		if ext == allowedExt {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("file extension not allowed: %s (allowed: %v)", ext, AllowedExtensions)
	}

	// Check for reserved names
	reservedNames := []string{"con", "prn", "aux", "nul", "com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9", "lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9"}
	baseName := strings.ToLower(filepath.Base(path))
	for _, reserved := range reservedNames {
		if baseName == reserved || strings.HasPrefix(baseName, reserved+".") {
			return fmt.Errorf("file name is reserved: %s", baseName)
		}
	}

	return nil
}

// ValidateContentSize validates content size against security limits
func ValidateContentSize(size int64) error {
	if size <= 0 {
		return errors.New("content size must be positive")
	}
	if size > MaxContentSize {
		return fmt.Errorf("content size too large: %d bytes (maximum %d bytes)", size, MaxContentSize)
	}
	return nil
}
