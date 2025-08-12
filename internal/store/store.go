package store

import (
    "bytes"
    "encoding/json"
    "errors"
    "path/filepath"
    "strings"
    "time"

    "github.com/dgraph-io/badger/v4"
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
            return err
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

func (s *Store) SetHead(siteID string, seq uint64, headCID string) error {
    return s.db.Update(func(txn *badger.Txn) error {
        var tmp [8]byte
        for i := 0; i < 8; i++ {
            tmp[7-i] = byte(seq >> (8 * i))
        }
        if err := txn.Set([]byte("headseq:"+siteID), tmp[:]); err != nil {
            return err
        }
        return txn.Set([]byte("headcid:"+siteID), []byte(headCID))
    })
}

func (s *Store) GetHead(siteID string) (seq uint64, headCID string, err error) {
    err = s.db.View(func(txn *badger.Txn) error {
        get := func(key string) ([]byte, error) {
            it, e := txn.Get([]byte(key))
            if e != nil {
                return nil, e
            }
            var v []byte
            if e := it.Value(func(b []byte) error { v = append([]byte{}, b...); return nil }); e != nil {
                return nil, e
            }
            return v, nil
        }
        seqB, e1 := get("headseq:" + siteID)
        if e1 != nil {
            return e1
        }
        var s64 uint64
        for i := 0; i < 8; i++ {
            s64 = (s64 << 8) | uint64(seqB[i])
        }
        cidB, e2 := get("headcid:" + siteID)
        if e2 != nil {
            return e2
        }
        seq = s64
        headCID = string(cidB)
        return nil
    })
    return
}

func (s *Store) HasHead(siteID string) (bool, error) {
    err := s.db.View(func(txn *badger.Txn) error {
        _, e := txn.Get([]byte("headcid:" + siteID))
        return e
    })
    if errors.Is(err, badger.ErrKeyNotFound) {
        return false, nil
    }
    return err == nil, err
}

// Domain name system functions
func (s *Store) RegisterDomain(domain string, siteID string, ownerPub []byte) error {
    // Validate domain format: alphanumerical.alphanumerical
    if !isValidDomain(domain) {
        return errors.New("invalid domain format: must be alphanumerical.alphanumerical")
    }
    
    return s.db.Update(func(txn *badger.Txn) error {
        // Check if domain already exists
        if _, err := txn.Get([]byte("domain:" + domain)); err == nil {
            return errors.New("domain already registered")
        }
        
        // Store domain mapping
        domainInfo := map[string]interface{}{
            "domain":   domain,
            "siteID":   siteID,
            "ownerPub": ownerPub,
            "created":  time.Now().Unix(),
        }
        
        domainData, err := json.Marshal(domainInfo)
        if err != nil {
            return err
        }
        
        // Store domain -> siteID mapping
        if err := txn.Set([]byte("domain:"+domain), domainData); err != nil {
            return err
        }
        
        // Store reverse mapping: siteID -> domain
        if err := txn.Set([]byte("sitedomain:"+siteID), []byte(domain)); err != nil {
            return err
        }
        
        return nil
    })
}

func (s *Store) ResolveDomain(domain string) (string, error) {
    var siteID string
    err := s.db.View(func(txn *badger.Txn) error {
        it, err := txn.Get([]byte("domain:" + domain))
        if err != nil {
            return err
        }
        
        var domainInfo map[string]interface{}
        if err := it.Value(func(v []byte) error {
            return json.Unmarshal(v, &domainInfo)
        }); err != nil {
            return err
        }
        
        if id, ok := domainInfo["siteID"].(string); ok {
            siteID = id
        }
        return nil
    })
    
    if err != nil {
        return "", err
    }
    
    return siteID, nil
}

func (s *Store) GetDomainOwner(domain string) ([]byte, error) {
    var ownerPub []byte
    err := s.db.View(func(txn *badger.Txn) error {
        it, err := txn.Get([]byte("domain:" + domain))
        if err != nil {
            return err
        }
        
        var domainInfo map[string]interface{}
        if err := it.Value(func(v []byte) error {
            return json.Unmarshal(v, &domainInfo)
        }); err != nil {
            return err
        }
        
        if owner, ok := domainInfo["ownerPub"].([]byte); ok {
            ownerPub = owner
        }
        return nil
    })
    
    return ownerPub, err
}

func (s *Store) ListDomains() ([]string, error) {
    var domains []string
    err := s.db.View(func(txn *badger.Txn) error {
        opts := badger.DefaultIteratorOptions
        opts.PrefetchValues = false
        it := txn.NewIterator(opts)
        defer it.Close()
        
        for it.Seek([]byte("domain:")); it.ValidForPrefix([]byte("domain:")); it.Next() {
            item := it.Item()
            k := item.Key()
            domain := string(k[len("domain:"):])
            domains = append(domains, domain)
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


