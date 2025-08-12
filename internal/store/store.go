package store

import (
    "bytes"
    "errors"
    "fmt"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "betanet/internal/core"

    "github.com/dgraph-io/badger/v4"
    "github.com/fxamacker/cbor/v2"
)

type Store struct {
    db *badger.DB
}

func Open(dir string) (*Store, error) {
    db, err := badger.Open(badger.DefaultOptions(filepath.Clean(dir)))
    if err != nil {
        return nil, err
    }
    return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) PutRecord(cid string, data []byte) error {
    return s.db.Update(func(txn *badger.Txn) error {
        return txn.Set([]byte("record:"+cid), data)
    })
}

func (s *Store) GetRecord(cid string) ([]byte, error) {
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
    return s.db.Update(func(txn *badger.Txn) error {
        return txn.Delete([]byte("record:" + cid))
    })
}

// ResolveRecordCID resolves a CID prefix to the full record CID, ensuring uniqueness.
func (s *Store) ResolveRecordCID(prefix string) (string, error) {
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
            if !bytes.HasPrefix(k, []byte("record:")) { continue }
            full := string(k[len("record:"):])
            found = full
            count++
            if count > 1 { break }
        }
        return nil
    })
    if err != nil { return "", err }
    if count == 0 { return "", badger.ErrKeyNotFound }
    if count > 1 { return "", errors.New("ambiguous record CID prefix") }
    return found, nil
}

func (s *Store) PutContent(cid string, data []byte) error {
    return s.db.Update(func(txn *badger.Txn) error {
        return txn.Set([]byte("content:"+cid), data)
    })
}

func (s *Store) GetContent(cid string) ([]byte, error) {
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
        for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
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
            break
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
            if !bytes.HasPrefix(k, []byte("content:")) { continue }
            full := string(k[len("content:"):])
            found = full
            count++
            if count > 1 { break }
        }
        return nil
    })
    if err != nil { return "", err }
    if count == 0 { return "", badger.ErrKeyNotFound }
    if count > 1 { return "", errors.New("ambiguous content CID prefix") }
    return found, nil
}

// Domain resolution methods

func (s *Store) PutDomain(domain string, siteID string) error {
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

// Helper function to validate domain format
func isValidDomain(domain string) bool {
    // Must be in format: alphanumerical.alphanumerical
    parts := strings.Split(domain, ".")
    if len(parts) != 2 {
        return false
    }
    
    // Both parts must be alphanumeric and non-empty
    for _, part := range parts {
        if len(part) == 0 {
            return false
        }
        for _, char := range part {
            if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
                return false
            }
        }
    }
    
    return true
}


