package main

import (
    "context"
    "crypto/ed25519"
    "encoding/base64"
    "encoding/hex"
    "flag"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "betanet/internal/core"
    bncrypto "betanet/internal/crypto"
    "betanet/internal/p2p"
    "betanet/internal/store"
    "betanet/internal/wallet"

    "github.com/fxamacker/cbor/v2"
    peer "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"
)

func main() {
    log.SetFlags(log.LstdFlags | log.Lmicroseconds)

    if len(os.Args) < 2 {
        usage()
        return
    }
    switch os.Args[1] {
    case "init-key":
        initKey()
    case "run":
        runNode()
    case "publish":
        publish()
    case "head":
        head()
    case "get-record":
        getRecord()
    case "get-content":
        getContent()
    case "delete":
        deleteCmd()
    case "browse":
        browse()
    default:
        usage()
    }
}

func usage() {
    fmt.Println("betanet-node commands:")
    fmt.Println("  init-key -key /path/key.b64")
    fmt.Println("  run -data /path/db -listen /ip4/0.0.0.0/tcp/4001 -bootstrap <ma1,ma2,...>")
    fmt.Println("  publish -key /path/key.b64 -data /path/db -content /path/file [-bootstrap ...]")
    fmt.Println("  head -key /path/key.b64 -data /path/db")
    fmt.Println("  get-record -data /path/db -recCID <hex>")
    fmt.Println("  get-content -data /path/db -contentCID <hex> [-out FILE] [-decrypt-pass \"...\"]")
    fmt.Println("  delete -key /path/key.b64 -data /path/db [-recCID <hex>] [-contentCID <hex>]")
    fmt.Println("  browse -data /path/db -site <siteID> [-listen MA] [-bootstrap MA[,MA...]] [-out FILE] [-decrypt-pass \"...\"]")
}

func initKey() {
    fs := flag.NewFlagSet("init-key", flag.ExitOnError)
    keyPath := fs.String("key", "site-key.b64", "path to write base64 ed25519 private key")
    _ = fs.Parse(os.Args[2:])
    pub, priv, err := bncrypto.GenerateSiteKey()
    if err != nil { log.Fatal(err) }
    if err := os.WriteFile(*keyPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0600); err != nil { log.Fatal(err) }
    fmt.Printf("Wrote %s\nSite pub: %x\nSiteID: %s\n", *keyPath, pub, core.SiteIDFromPub(pub))
}

func runNode() {
    fs := flag.NewFlagSet("run", flag.ExitOnError)
    data := fs.String("data", "./data", "data directory")
    listen := fs.String("listen", "/ip4/0.0.0.0/tcp/4001", "libp2p listen multiaddr")
    bootstrap := fs.String("bootstrap", "", "comma-separated peer multiaddrs")
    _ = fs.Parse(os.Args[2:])

    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    ctx := context.Background()
    node, err := p2p.New(ctx, db, *listen, strings.Split(*bootstrap, ","))
    if err != nil { log.Fatal(err) }
    if err := node.Start(ctx); err != nil { log.Fatal(err) }

    log.Printf("node running, listen=%s", *listen)
    // Print libp2p addresses with peer ID to help bootstrap
    if node.Host != nil {
        pid := node.Host.ID().String()
        for _, a := range node.Host.Addrs() {
            fmt.Printf("addr: %s/p2p/%s\n", a.String(), pid)
        }
    }
    select {}
}

func publish() {
    fs := flag.NewFlagSet("publish", flag.ExitOnError)
    keyPath := fs.String("key", "site-key.b64", "site key (base64 ed25519 priv)")
    data := fs.String("data", "./data", "data directory")
    contentPath := fs.String("content", "", "file with content bytes")
    listen := fs.String("listen", "/ip4/0.0.0.0/tcp/0", "listen (ephemeral)")
    bootstrap := fs.String("bootstrap", "", "comma-separated peer multiaddrs")
    _ = fs.Parse(os.Args[2:])
    if *contentPath == "" { log.Fatal("missing -content") }

    raw, err := os.ReadFile(*keyPath)
    if err != nil { log.Fatal(err) }
    privBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
    if err != nil { log.Fatal(err) }
    priv := ed25519.PrivateKey(privBytes)
    pub := priv.Public().(ed25519.PublicKey)

    cnt, err := os.ReadFile(*contentPath)
    if err != nil { log.Fatal(err) }

    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    node, err := p2p.New(ctx, db, *listen, strings.Split(*bootstrap, ","))
    if err != nil { log.Fatal(err) }
    if err := node.Start(ctx); err != nil { log.Fatal(err) }

    siteID := core.SiteIDFromPub(pub)
    var seq uint64 = 1
    prevCID := ""
    if has, _ := db.HasHead(siteID); has {
        s, headCID, err := db.GetHead(siteID)
        if err != nil { log.Fatal(err) }
        seq = s + 1
        prevCID = headCID
    }

    env, recCID, err := node.BuildUpdate(ed25519.PrivateKey(priv), ed25519.PublicKey(pub), cnt, seq, prevCID)
    if err != nil { log.Fatal(err) }

    if err := db.PutRecord(recCID, env.Record); err != nil { log.Fatal(err) }
    if len(env.Content) > 0 {
        if err := db.PutContent(core.CIDForContent(env.Content), env.Content); err != nil { log.Fatal(err) }
    }
    if err := db.SetHead(siteID, seq, recCID); err != nil { log.Fatal(err) }

    if err := node.BroadcastUpdate(ctx, *env); err != nil { log.Fatal(err) }

    fmt.Printf("Published seq=%d recCID=%s contentCID=%s siteID=%s\n",
        seq, shortStr(recCID), shortStr(core.CIDForContent(cnt)), siteID[:12])
}

func head() {
    fs := flag.NewFlagSet("head", flag.ExitOnError)
    keyPath := fs.String("key", "site-key.b64", "site key (base64 ed25519 priv)")
    data := fs.String("data", "./data", "data directory")
    _ = fs.Parse(os.Args[2:])

    raw, err := os.ReadFile(*keyPath)
    if err != nil { log.Fatal(err) }
    privBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
    if err != nil { log.Fatal(err) }
    priv := ed25519.PrivateKey(privBytes)
    pub := priv.Public().(ed25519.PublicKey)

    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    siteID := core.SiteIDFromPub(pub)
    if has, _ := db.HasHead(siteID); !has {
        fmt.Println("no head")
        return
    }
    seq, headCID, err := db.GetHead(siteID)
    if err != nil { log.Fatal(err) }
    fmt.Printf("siteID=%s seq=%d headCID=%s\n", siteID[:12], seq, shortStr(headCID))
}

func shortStr(s string) string {
    if len(s) > 12 { return s[:12] }
    return s
}

func getRecord() {
    fs := flag.NewFlagSet("get-record", flag.ExitOnError)
    data := fs.String("data", "./data", "data directory")
    recCID := fs.String("recCID", "", "record CID or prefix (hex)")
    _ = fs.Parse(os.Args[2:])
    if *recCID == "" { log.Fatal("missing -recCID") }

    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    cid := *recCID
    if len(cid) < 64 {
        full, err := db.ResolveRecordCID(cid)
        if err != nil { log.Fatal(err) }
        cid = full
    }
    b, err := db.GetRecord(cid)
    if err != nil { log.Fatal(err) }
    var rec core.UpdateRecord
    dec, _ := cbor.DecOptions{}.DecMode()
    if err := dec.Unmarshal(b, &rec); err != nil { log.Fatal(err) }
    fmt.Printf("Record %s\n", shortStr(cid))
    fmt.Printf("  Version: %s\n", rec.Version)
    fmt.Printf("  SitePub: %s\n", hex.EncodeToString(rec.SitePub))
    fmt.Printf("  Seq: %d\n", rec.Seq)
    fmt.Printf("  PrevCID: %s\n", rec.PrevCID)
    fmt.Printf("  ContentCID: %s\n", rec.ContentCID)
    fmt.Printf("  TS: %d\n", rec.TS)
    fmt.Printf("  UpdatePub: %s\n", hex.EncodeToString(rec.UpdatePub))
}

func getContent() {
    fs := flag.NewFlagSet("get-content", flag.ExitOnError)
    data := fs.String("data", "./data", "data directory")
    cid := fs.String("contentCID", "", "content CID or prefix (hex)")
    out := fs.String("out", "", "output file (default stdout)")
    decPass := fs.String("decrypt-pass", "", "passphrase to decrypt content (optional)")
    _ = fs.Parse(os.Args[2:])
    if *cid == "" { log.Fatal("missing -contentCID") }

    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    contentCID := *cid
    if len(contentCID) < 64 {
        full, err := db.ResolveContentCID(contentCID)
        if err != nil { log.Fatal(err) }
        contentCID = full
    }
    b, err := db.GetContent(contentCID)
    if err != nil { log.Fatal(err) }
    if *decPass != "" {
        pt, err := wallet.DecryptContent(*decPass, b)
        if err != nil { log.Fatal(err) }
        b = pt
    }
    if *out == "" {
        os.Stdout.Write(b)
        return
    }
    if err := os.WriteFile(*out, b, 0644); err != nil { log.Fatal(err) }
    fmt.Printf("Wrote %s (%d bytes)\n", *out, len(b))
}

func deleteCmd() {
    fs := flag.NewFlagSet("delete", flag.ExitOnError)
    keyPath := fs.String("key", "site-key.b64", "site key (base64 ed25519 priv)")
    data := fs.String("data", "./data", "data directory")
    recCID := fs.String("recCID", "", "record CID to delete")
    contentCID := fs.String("contentCID", "", "content CID to delete")
    listen := fs.String("listen", "/ip4/0.0.0.0/tcp/0", "listen (ephemeral)")
    bootstrap := fs.String("bootstrap", "", "comma-separated peer multiaddrs")
    _ = fs.Parse(os.Args[2:])
    if *recCID == "" && *contentCID == "" { log.Fatal("provide -recCID and/or -contentCID") }

    raw, err := os.ReadFile(*keyPath)
    if err != nil { log.Fatal(err) }
    privBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
    if err != nil { log.Fatal(err) }
    priv := ed25519.PrivateKey(privBytes)
    pub := priv.Public().(ed25519.PublicKey)

    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    node, err := p2p.New(ctx, db, *listen, strings.Split(*bootstrap, ","))
    if err != nil { log.Fatal(err) }
    if err := node.Start(ctx); err != nil { log.Fatal(err) }

    del := core.DeleteRecord{Version: "v1", SitePub: pub, TargetRec: *recCID, TargetCont: *contentCID, TS: time.Now().Unix()}
    pre := bncrypto.PreimageDelete(del.SitePub, del.TargetRec, del.TargetCont, del.TS)
    del.Sig = ed25519.Sign(priv, pre)

    if err := node.BroadcastDelete(ctx, del); err != nil { log.Fatal(err) }
    fmt.Println("Delete broadcasted")
}

func browse() {
    fs := flag.NewFlagSet("browse", flag.ExitOnError)
    data := fs.String("data", "./data", "data directory for session cache")
    listen := fs.String("listen", "/ip4/0.0.0.0/tcp/0", "listen addr")
    bootstrap := fs.String("bootstrap", "", "comma-separated peers (optional)")
    site := fs.String("site", "", "siteID (hex)")
    out := fs.String("out", "", "write fetched content to file")
    decPass := fs.String("decrypt-pass", "", "optional passphrase to decrypt content")
    _ = fs.Parse(os.Args[2:])
    if *site == "" { log.Fatal("missing -site") }

    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    node, err := p2p.New(ctx, db, *listen, strings.Split(*bootstrap, ","))
    if err != nil { log.Fatal(err) }
    if err := node.Start(ctx); err != nil { log.Fatal(err) }

    var target *peer.AddrInfo
    
    // First try auto-discovery via mDNS (for LAN peers)
    log.Printf("🔍 Searching for peers via mDNS...")
    if best, err := node.DiscoverBestPeer(ctx, 3*time.Second); err == nil {
        target = best
        log.Printf("✅ Found LAN peer: %s", target.ID)
    } else {
        log.Printf("⚠️  mDNS discovery failed: %v", err)
    }
    
    // If mDNS failed, try localhost discovery (zero-config for local development)
    if target == nil {
        log.Printf("🔍 Searching localhost for betanet nodes...")
        if localhost, err := node.DiscoverLocalhostPeer(ctx); err == nil {
            target = localhost
            log.Printf("✅ Found localhost peer: %s", target.ID)
        } else {
            log.Printf("⚠️  Localhost discovery failed: %v", err)
        }
    }
    
    // If both auto-discovery methods failed, try manual bootstrap
    if target == nil {
        bs := strings.TrimSpace(*bootstrap)
        if bs != "" {
            log.Printf("🔍 Trying manual bootstrap address...")
            first := strings.Split(bs, ",")[0]
            if m, err := ma.NewMultiaddr(first); err == nil {
                if ai, err := peer.AddrInfoFromP2pAddr(m); err == nil { 
                    target = ai 
                    log.Printf("✅ Bootstrap successful: %s", target.ID)
                }
            }
            if target == nil {
                log.Printf("❌ Bootstrap failed: invalid address format")
            }
        } else {
            // Try default localhost addresses as fallback
            log.Printf("🔗 Trying default localhost addresses...")
            defaultAddrs := []string{
                "/ip4/127.0.0.1/tcp/4001",
                "/ip4/127.0.0.1/tcp/4002", 
                "/ip4/127.0.0.1/tcp/4003",
            }
            for _, addr := range defaultAddrs {
                log.Printf("🔍 Trying %s...", addr)
                if _, err := ma.NewMultiaddr(addr); err == nil {
                    // We can't get peer ID without connecting, so we'll just log the attempt
                    log.Printf("⚠️  Note: Default addresses need peer ID. Use 'addr:' line from running node")
                    break
                }
            }
        }
    }
    
    if target == nil { 
        log.Fatal("❌ No betanet nodes found!\n\n" +
                 "🔧 Quick Fix:\n" +
                 "1. Start a node: ./bin/betanet-node run -data /tmp/node -listen /ip4/0.0.0.0/tcp/4001\n" +
                 "2. Copy the 'addr:' line and use: -bootstrap <ADDR>\n" +
                 "3. Example: -bootstrap /ip4/127.0.0.1/tcp/4001/p2p/12D3Koo...\n\n" +
                 "💡 The bootstrap address is auto-filled in the GUI browser!")
    }

    // Proactively connect for clearer diagnostics before making requests
    {
        cctx, ccancel := context.WithTimeout(ctx, 5*time.Second)
        defer ccancel()
        if err := node.Host.Connect(cctx, *target); err != nil {
            if strings.Contains(strings.ToLower(err.Error()), "dial backoff") {
                log.Fatalf("connect failed: %v\nHint: the peer at the provided address is unreachable. Ensure you're using the exact 'addr:' line printed by the node, that it is still running, and that you used 127.0.0.1 if on the same host.", err)
            }
            log.Fatalf("connect failed: %v", err)
        }
    }

    seq, headCID, contentCID, err := node.RequestHead(ctx, *target, *site)
    if err != nil { log.Fatal(err) }
    fmt.Printf("Head: seq=%d headCID=%s contentCID=%s\n", seq, headCID, contentCID)

    content, err := node.RequestContent(ctx, *target, contentCID)
    if err != nil { log.Fatal(err) }
    if *decPass != "" {
        if pt, derr := wallet.DecryptContent(*decPass, content); derr == nil { content = pt }
    }
    if *out == "" {
        os.Stdout.Write(content)
        if len(content) == 0 || content[len(content)-1] != '\n' { fmt.Println() }
        return
    }
    if err := os.WriteFile(*out, content, 0644); err != nil { log.Fatal(err) }
    fmt.Printf("Wrote %s (%d bytes)\n", *out, len(content))
}

// no extra helpers


