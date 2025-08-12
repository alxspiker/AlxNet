package core

import (
    "crypto/sha256"
    "encoding/hex"
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

// DeleteRecord authorizes deletion of a specific record/content for a site.
type DeleteRecord struct {
    Version    string `cbor:"0,keyasint"`
    SitePub    []byte `cbor:"1,keyasint"`
    TargetRec  string `cbor:"2,keyasint"` // record CID to delete (optional)
    TargetCont string `cbor:"3,keyasint"` // content CID to delete (optional)
    TS         int64  `cbor:"4,keyasint"`
    Sig        []byte `cbor:"5,keyasint"` // Ed25519 by SitePriv over PreimageDelete
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


