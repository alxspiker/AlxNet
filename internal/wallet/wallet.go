package wallet

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "errors"
    "io"
    "os"
    "path/filepath"
    "strings"

    "betanet/internal/core"

    "golang.org/x/crypto/argon2"
    "golang.org/x/crypto/chacha20poly1305"
    "golang.org/x/crypto/hkdf"
    "github.com/tyler-smith/go-bip39"
)

const (
    walletVersion = 1
    kdfName       = "argon2id"
    adWallet      = "bn-wallet-v1"
    adContent     = "bn-content-v1"
    contentHdr    = "BNE1" // 4 bytes
)

type SiteMeta struct {
    Label       string `json:"label"`
    SiteID      string `json:"site_id"`
    SitePubHex  string `json:"site_pub"`
    Seq         uint64 `json:"seq"`
    HeadRecCID  string `json:"head_rec_cid"`
    ContentCID  string `json:"content_cid"`
}

type Wallet struct {
    Version int                  `json:"v"`
    Sites   map[string]*SiteMeta `json:"sites"` // key = label
}

type encFile struct {
    Version    int    `json:"v"`
    KDF        string `json:"kdf"`
    SaltB64    string `json:"salt"`
    T          uint32 `json:"t"`
    MiB        uint32 `json:"m"`
    P          uint8  `json:"p"`
    NonceB64   string `json:"nonce"`
    CipherB64  string `json:"ct"`
}

func NewMnemonic() (string, error) {
    entropy, err := bip39.NewEntropy(256)
    if err != nil { return "", err }
    return bip39.NewMnemonic(entropy)
}

func masterKeyFromMnemonic(mnemonic string) ([]byte, error) {
    seed := bip39.NewSeed(mnemonic, "")
    h := hkdf.New(sha256.New, seed, []byte("bn-wallet-v1"), []byte("master"))
    key := make([]byte, 32)
    if _, err := io.ReadFull(h, key); err != nil { return nil, err }
    return key, nil
}

func DeriveSiteKey(master []byte, label string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
    h := hkdf.New(sha256.New, master, []byte("bn-site"), []byte(strings.ToLower(label)))
    seed := make([]byte, 32)
    if _, err := io.ReadFull(h, seed); err != nil { return nil, nil, err }
    priv := ed25519.NewKeyFromSeed(seed)
    return priv.Public().(ed25519.PublicKey), priv, nil
}

func EncryptWallet(w *Wallet, mnemonic string) ([]byte, error) {
    raw, err := json.Marshal(w)
    if err != nil { return nil, err }

    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil { return nil, err }

    t, mMiB, p := uint32(2), uint32(64), uint8(4)
    key := argon2.IDKey([]byte(mnemonic), salt, t, mMiB*1024, uint8(p), 32)

    aead, err := chacha20poly1305.NewX(key)
    if err != nil { return nil, err }
    nonce := make([]byte, chacha20poly1305.NonceSizeX)
    if _, err := rand.Read(nonce); err != nil { return nil, err }

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
    return json.MarshalIndent(out, "", "  ")
}

func DecryptWallet(encBytes []byte, mnemonic string) (*Wallet, error) {
    var ef encFile
    if err := json.Unmarshal(encBytes, &ef); err != nil { return nil, err }
    if ef.KDF != kdfName || ef.Version != walletVersion {
        return nil, errors.New("unsupported wallet format")
    }
    salt, err := base64.StdEncoding.DecodeString(ef.SaltB64)
    if err != nil { return nil, err }
    nonce, err := base64.StdEncoding.DecodeString(ef.NonceB64)
    if err != nil { return nil, err }
    ct, err := base64.StdEncoding.DecodeString(ef.CipherB64)
    if err != nil { return nil, err }

    key := argon2.IDKey([]byte(mnemonic), salt, ef.T, ef.MiB*1024, uint8(ef.P), 32)
    aead, err := chacha20poly1305.NewX(key)
    if err != nil { return nil, err }
    raw, err := aead.Open(nil, nonce, ct, []byte(adWallet))
    if err != nil { return nil, errors.New("bad mnemonic or corrupted wallet") }
    var w Wallet
    if err := json.Unmarshal(raw, &w); err != nil { return nil, err }
    return &w, nil
}

func Save(path string, bytes []byte) error {
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
    return os.WriteFile(path, bytes, 0o600)
}

func Load(path string) ([]byte, error) {
    return os.ReadFile(path)
}

func New() *Wallet {
    return &Wallet{Version: walletVersion, Sites: map[string]*SiteMeta{}}
}

func (w *Wallet) EnsureSite(master []byte, label string) (*SiteMeta, ed25519.PublicKey, ed25519.PrivateKey, error) {
    pub, priv, err := DeriveSiteKey(master, label)
    if err != nil { return nil, nil, nil, err }
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
    meta := &SiteMeta{
        Label:      label,
        SiteID:     siteID,
        SitePubHex: strings.ToLower(hexOf(pub)),
    }
    w.Sites[label] = meta
    return meta, pub, priv, nil
}

// Encrypt content with passphrase. Output: "BNE1" || salt(16) || nonce(24) || ciphertext
func EncryptContent(passphrase string, plaintext []byte) ([]byte, error) {
    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil { return nil, err }
    key := argon2.IDKey([]byte(passphrase), salt, 2, 64*1024, 4, 32)
    aead, err := chacha20poly1305.NewX(key)
    if err != nil { return nil, err }
    nonce := make([]byte, chacha20poly1305.NonceSizeX)
    if _, err := rand.Read(nonce); err != nil { return nil, err }
    ct := aead.Seal(nil, nonce, plaintext, []byte(adContent))
    out := make([]byte, 0, 4+16+len(nonce)+len(ct))
    out = append(out, contentHdr...)
    out = append(out, salt...)
    out = append(out, nonce...)
    out = append(out, ct...)
    return out, nil
}

func DecryptContent(passphrase string, blob []byte) ([]byte, error) {
    if len(blob) < 4+16+chacha20poly1305.NonceSizeX { return nil, errors.New("invalid blob") }
    if string(blob[:4]) != contentHdr { return nil, errors.New("bad header") }
    salt := blob[4:20]
    nonce := blob[20 : 20+chacha20poly1305.NonceSizeX]
    ct := blob[20+chacha20poly1305.NonceSizeX:]
    key := argon2.IDKey([]byte(passphrase), salt, 2, 64*1024, 4, 32)
    aead, err := chacha20poly1305.NewX(key)
    if err != nil { return nil, err }
    return aead.Open(nil, nonce, ct, []byte(adContent))
}


