package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
)

func GenerateSiteKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

func GenerateUpdateKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// PreimageLink binds SitePub to UpdatePub for a given (seq, prevCID, contentCID, ts)
// This is signed by the long-term Site private key.
func PreimageLink(sitePub, updatePub []byte, seq uint64, prevCID, contentCID string, ts int64) []byte {
	h := sha256.New()
	h.Write([]byte("bn-link-v1"))
	h.Write(sitePub)
	h.Write(updatePub)
	var tmp [8]byte
	for i := 0; i < 8; i++ {
		tmp[7-i] = byte(seq >> (8 * i))
	}
	h.Write(tmp[:])
	h.Write([]byte(prevCID))
	h.Write([]byte(contentCID))
	var t [8]byte
	u := uint64(ts)
	for i := 0; i < 8; i++ {
		t[7-i] = byte(u >> (8 * i))
	}
	h.Write(t[:])
	return h.Sum(nil)
}

// PreimageUpdate is signed by the per-update ephemeral key over the record
// bytes with UpdateSig field cleared, then prefixed with a domain tag.
func PreimageUpdate(recordBytes []byte) []byte {
	sum := sha256.Sum256(append([]byte("bn-update-v1"), recordBytes...))
	return sum[:]
}

// PreimageDelete is signed by the Site private key to authorize deletion of a
// specific record/content. Fields are concatenated in fixed order under a
// domain separation tag.
func PreimageDelete(sitePub []byte, targetRecCID, targetContentCID string, ts int64) []byte {
	h := sha256.New()
	h.Write([]byte("bn-del-v1"))
	h.Write(sitePub)
	h.Write([]byte(targetRecCID))
	h.Write([]byte(targetContentCID))
	var t [8]byte
	u := uint64(ts)
	for i := 0; i < 8; i++ {
		t[7-i] = byte(u >> (8 * i))
	}
	h.Write(t[:])
	return h.Sum(nil)
}
