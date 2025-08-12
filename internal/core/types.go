package core

import (
    "crypto/sha256"
    "encoding/hex"
    "strings"
    "time"

    "github.com/fxamacker/cbor/v2"
)

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

// WebsiteManifest represents a multi-file website with references to all its files
type WebsiteManifest struct {
    Version     string            `cbor:"0,keyasint"`
    SitePub     []byte            `cbor:"1,keyasint"` // 32B ed25519 pub
    Seq         uint64            `cbor:"2,keyasint"`
    PrevCID     string            `cbor:"3,keyasint"` // hex sha256 of previous manifest
    TS          int64             `cbor:"4,keyasint"` // unix seconds
    MainFile    string            `cbor:"5,keyasint"` // path to main entry point (e.g., "index.html")
    Files       map[string]string `cbor:"6,keyasint"` // path -> content CID mapping
    UpdatePub   []byte            `cbor:"7,keyasint"` // 32B ed25519 pub (ephemeral per update)
    LinkSig     []byte            `cbor:"8,keyasint"` // sig by SitePriv over link preimage
    UpdateSig   []byte            `cbor:"9,keyasint"` // sig by UpdatePriv over manifest preimage
}

// FileRecord represents a file within a multi-file website
type FileRecord struct {
    Version     string `cbor:"0,keyasint"`
    SitePub     []byte `cbor:"1,keyasint"` // 32B ed25519 pub
    Path        string `cbor:"2,keyasint"` // file path within website (e.g., "styles/main.css")
    ContentCID  string `cbor:"3,keyasint"` // hex sha256 of file content
    MimeType    string `cbor:"4,keyasint"` // MIME type of the file
    TS          int64  `cbor:"5,keyasint"` // unix seconds
    UpdatePub   []byte `cbor:"6,keyasint"` // 32B ed25519 pub (ephemeral per update)
    LinkSig     []byte `cbor:"7,keyasint"` // sig by SitePriv over link preimage
    UpdateSig   []byte `cbor:"8,keyasint"` // sig by UpdatePriv over file record preimage
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
    SiteID      string                    `json:"site_id"`
    MainFile    string                    `json:"main_file"`
    Files       map[string]WebsiteFileInfo `json:"files"`
    FileCount   int                       `json:"file_count"`
    LastUpdated time.Time                 `json:"last_updated"`
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


