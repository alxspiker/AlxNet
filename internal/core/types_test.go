package core

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestUpdateRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		record  UpdateRecord
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid record",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: false,
		},
		{
			name: "missing version",
			record: UpdateRecord{
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "invalid site public key length",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 16), // Too short
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "invalid site public key length",
		},
		{
			name: "sequence number zero",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        0, // Invalid
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "sequence number out of range",
		},
		{
			name: "invalid timestamp",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         0, // Invalid
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "invalid timestamp",
		},
		{
			name: "future timestamp",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         time.Now().Unix() + 7200, // 2 hours in future
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "timestamp too far in future",
		},
		{
			name: "invalid update public key length",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 16), // Too short
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "invalid update public key length",
		},
		{
			name: "invalid link signature length",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 32), // Too short
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "invalid link signature length",
		},
		{
			name: "invalid update signature length",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "def456",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 32), // Too short
			},
			wantErr: true,
			errMsg:  "invalid update signature length",
		},
		{
			name: "missing content CID",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "", // Missing
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "content CID is required",
		},
		{
			name: "invalid hex string in prev CID",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "invalid-hex!", // Invalid hex
				ContentCID: "def456",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "invalid previous CID format",
		},
		{
			name: "invalid hex string in content CID",
			record: UpdateRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Seq:        1,
				PrevCID:    "abc123",
				ContentCID: "invalid-hex!", // Invalid hex
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "invalid content CID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateRecord.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("UpdateRecord.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestWebsiteManifestValidation(t *testing.T) {
	tests := []struct {
		name     string
		manifest WebsiteManifest
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid manifest",
			manifest: WebsiteManifest{
				Version:   "1.0",
				SitePub:   make([]byte, 32),
				Seq:       1,
				PrevCID:   "abc123",
				TS:        time.Now().Unix(),
				MainFile:  "index.html",
				Files:     map[string]string{"index.html": "def456"},
				UpdatePub: make([]byte, 32),
				LinkSig:   make([]byte, 64),
				UpdateSig: make([]byte, 64),
			},
			wantErr: false,
		},
		{
			name: "missing main file",
			manifest: WebsiteManifest{
				Version:   "1.0",
				SitePub:   make([]byte, 32),
				Seq:       1,
				PrevCID:   "abc123",
				TS:        time.Now().Unix(),
				MainFile:  "", // Missing
				Files:     map[string]string{"index.html": "def456"},
				UpdatePub: make([]byte, 32),
				LinkSig:   make([]byte, 64),
				UpdateSig: make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "main file is required",
		},
		{
			name: "too many files",
			manifest: WebsiteManifest{
				Version:  "1.0",
				SitePub:  make([]byte, 32),
				Seq:      1,
				PrevCID:  "abc123",
				TS:       time.Now().Unix(),
				MainFile: "index.html",
				Files: func() map[string]string {
					files := make(map[string]string)
					for i := 0; i <= MaxFileCount; i++ {
						files[fmt.Sprintf("file%d.txt", i)] = "content"
					}
					return files
				}(),
				UpdatePub: make([]byte, 32),
				LinkSig:   make([]byte, 64),
				UpdateSig: make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "too many files",
		},
		{
			name: "invalid file path",
			manifest: WebsiteManifest{
				Version:   "1.0",
				SitePub:   make([]byte, 32),
				Seq:       1,
				PrevCID:   "abc123",
				TS:        time.Now().Unix(),
				MainFile:  "index.html",
				Files:     map[string]string{"../../../etc/passwd": "def456"}, // Invalid path
				UpdatePub: make([]byte, 32),
				LinkSig:   make([]byte, 64),
				UpdateSig: make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "invalid file path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WebsiteManifest.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("WebsiteManifest.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestFileRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		record  FileRecord
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid file record",
			record: FileRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Path:       "css/style.css",
				ContentCID: "def456",
				MimeType:   "text/css",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: false,
		},
		{
			name: "missing file path",
			record: FileRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Path:       "", // Missing
				ContentCID: "def456",
				MimeType:   "text/css",
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "file path is required",
		},
		{
			name: "invalid MIME type",
			record: FileRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				Path:       "css/style.css",
				ContentCID: "def456",
				MimeType:   "", // Missing
				TS:         time.Now().Unix(),
				UpdatePub:  make([]byte, 32),
				LinkSig:    make([]byte, 64),
				UpdateSig:  make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "MIME type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FileRecord.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("FileRecord.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestDeleteRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		record  DeleteRecord
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid delete record with record target",
			record: DeleteRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				TargetRec:  "abc123",
				TargetCont: "",
				TS:         time.Now().Unix(),
				Sig:        make([]byte, 64),
			},
			wantErr: false,
		},
		{
			name: "valid delete record with content target",
			record: DeleteRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				TargetRec:  "",
				TargetCont: "def456",
				TS:         time.Now().Unix(),
				Sig:        make([]byte, 64),
			},
			wantErr: false,
		},
		{
			name: "missing both targets",
			record: DeleteRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				TargetRec:  "", // Missing
				TargetCont: "", // Missing
				TS:         time.Now().Unix(),
				Sig:        make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "at least one target",
		},
		{
			name: "invalid target record CID",
			record: DeleteRecord{
				Version:    "1.0",
				SitePub:    make([]byte, 32),
				TargetRec:  "invalid-hex!", // Invalid hex
				TargetCont: "",
				TS:         time.Now().Unix(),
				Sig:        make([]byte, 64),
			},
			wantErr: true,
			errMsg:  "invalid target record CID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteRecord.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("DeleteRecord.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{"valid HTML file", "index.html", false, ""},
		{"valid CSS file", "css/style.css", false, ""},
		{"valid JS file", "js/app.js", false, ""},
		{"valid image file", "images/logo.png", false, ""},
		{"valid nested path", "assets/fonts/roboto.woff2", false, ""},
		{"empty path", "", true, "file path cannot be empty"},
		{"path too long", string(make([]byte, MaxPathLength+1)), true, "file path too long"},
		{"path traversal attempt", "../../../etc/passwd", true, "contains path traversal elements"},
		{"double slash", "css//style.css", true, "contains path traversal elements"},
		{"absolute path", "/home/user/file.html", true, "absolute file paths are not allowed"},
		{"no extension", "README", true, "file must have a valid extension"},
		{"invalid extension", "script.bat", true, "file extension not allowed"},
		{"reserved name", "con.txt", true, "file name is reserved"},
		{"reserved name with extension", "lpt1.html", true, "file name is reserved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateFilePath(%q) error message = %v, want %v", tt.path, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestValidateContentSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
		errMsg  string
	}{
		{"valid size", 1024, false, ""},
		{"zero size", 0, true, "content size must be positive"},
		{"negative size", -1024, true, "content size must be positive"},
		{"exactly at limit", MaxContentSize, false, ""},
		{"over limit", MaxContentSize + 1, true, "content size too large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContentSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContentSize(%d) error = %v, wantErr %v", tt.size, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateContentSize(%d) error message = %v, want %v", tt.size, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestIsValidHexString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty string", "", false},
		{"odd length", "abc", false},
		{"valid hex", "abcdef1234567890", true},
		{"valid hex uppercase", "ABCDEF1234567890", true},
		{"invalid characters", "abcdefgh", false},
		{"mixed case", "AbCdEf1234567890", true},
		{"single character", "a", false},
		{"two characters", "ab", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidHexString(tt.s); got != tt.want {
				t.Errorf("isValidHexString(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestIsValidMimeType(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		want     bool
	}{
		{"valid HTML", "text/html", true},
		{"valid CSS", "text/css", true},
		{"valid JS", "application/javascript", true},
		{"valid PNG", "image/png", true},
		{"valid JPEG", "image/jpeg", true},
		{"valid JSON", "application/json", true},
		{"valid XML", "application/xml", true},
		{"valid font", "font/woff2", true},
		{"valid webp", "image/webp", true},
		{"empty string", "", false},
		{"too long", string(make([]byte, 128)), false},
		{"custom valid", "application/x-custom", true},
		{"invalid format", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidMimeType(tt.mimeType); got != tt.want {
				t.Errorf("isValidMimeType(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}

func TestFileTypeDetection(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"HTML file", "index.html", true},
		{"HTML file uppercase", "INDEX.HTML", true},
		{"HTML file short", "page.htm", true},
		{"CSS file", "style.css", true},
		{"CSS file uppercase", "STYLE.CSS", true},
		{"JS file", "script.js", true},
		{"JS file uppercase", "SCRIPT.JS", true},
		{"PNG image", "logo.png", true},
		{"JPG image", "photo.jpg", true},
		{"JPEG image", "photo.jpeg", true},
		{"GIF image", "animation.gif", true},
		{"SVG image", "icon.svg", true},
		{"ICO image", "favicon.ico", true},
		{"not HTML", "script.js", false},
		{"not CSS", "index.html", false},
		{"not JS", "style.css", false},
		{"not image", "document.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHTMLFile(tt.path); got != (tt.path == "index.html" || tt.path == "INDEX.HTML" || tt.path == "page.htm") {
				t.Errorf("IsHTMLFile(%q) = %v, want %v", tt.path, got, !got)
			}
			if got := IsCSSFile(tt.path); got != (tt.path == "style.css" || tt.path == "STYLE.CSS") {
				t.Errorf("IsCSSFile(%q) = %v, want %v", tt.path, got, !got)
			}
			if got := IsJSFile(tt.path); got != (tt.path == "script.js" || tt.path == "SCRIPT.JS") {
				t.Errorf("IsJSFile(%q) = %v, want %v", tt.path, got, !got)
			}
			if got := IsImageFile(tt.path); got != (strings.Contains(tt.path, ".png") || strings.Contains(tt.path, ".jpg") || strings.Contains(tt.path, ".jpeg") || strings.Contains(tt.path, ".gif") || strings.Contains(tt.path, ".svg") || strings.Contains(tt.path, ".ico")) {
				t.Errorf("IsImageFile(%q) = %v, want %v", tt.path, got, !got)
			}
		})
	}
}

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"HTML file", "index.html", "text/html"},
		{"CSS file", "style.css", "text/css"},
		{"JS file", "script.js", "application/javascript"},
		{"PNG image", "logo.png", "image/png"},
		{"JPG image", "photo.jpg", "image/jpeg"},
		{"JPEG image", "photo.jpeg", "image/jpeg"},
		{"GIF image", "animation.gif", "image/gif"},
		{"SVG image", "icon.svg", "image/svg+xml"},
		{"ICO image", "favicon.ico", "image/x-icon"},
		{"JSON file", "data.json", "application/json"},
		{"XML file", "config.xml", "application/xml"},
		{"Text file", "README.txt", "text/plain"},
		{"Markdown file", "docs.md", "text/markdown"},
		{"Unknown extension", "file.xyz", "application/octet-stream"},
		{"No extension", "README", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMimeType(tt.path); got != tt.want {
				t.Errorf("GetMimeType(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
