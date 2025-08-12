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
    "path/filepath"
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
    "encoding/json"
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
    case "publish-website":
        publishWebsite()
    case "add-file":
        addFile()
    case "list-website":
        listWebsite()
    case "get-website-info":
        getWebsiteInfo()
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
    fmt.Println("")
    fmt.Println("Multi-file Website Commands:")
    fmt.Println("  publish-website -key /path/key.b64 -data /path/db -dir /path/website [-main index.html] [-bootstrap ...]")
    fmt.Println("  add-file -key /path/key.b64 -data /path/db -path <filepath> -content /path/file [-bootstrap ...]")
    fmt.Println("  list-website -data /path/db -site <siteID>")
    fmt.Println("  get-website-info -data /path/db -site <siteID>")
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

    // Keep node running
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

// publishWebsite publishes a complete multi-file website
func publishWebsite() {
    fs := flag.NewFlagSet("publish-website", flag.ExitOnError)
    keyPath := fs.String("key", "", "path to base64 ed25519 private key")
    data := fs.String("data", "./data", "data directory")
    websiteDir := fs.String("dir", "", "path to website directory")
    mainFile := fs.String("main", "index.html", "main entry point file")
    bootstrap := fs.String("bootstrap", "", "comma-separated peer multiaddrs")
    _ = fs.Parse(os.Args[2:])

    if *keyPath == "" || *websiteDir == "" {
        log.Fatal("key and dir are required")
    }

    // Load private key
    keyData, err := os.ReadFile(*keyPath)
    if err != nil { log.Fatal(err) }
    privBytes, err := base64.StdEncoding.DecodeString(string(keyData))
    if err != nil { log.Fatal(err) }
    priv := ed25519.PrivateKey(privBytes)
    pub := priv.Public().(ed25519.PublicKey)

    // Open database
    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    // Check if main file exists
    mainFilePath := filepath.Join(*websiteDir, *mainFile)
    if _, err := os.Stat(mainFilePath); os.IsNotExist(err) {
        log.Fatalf("Main file %s does not exist", mainFilePath)
    }

    // Start node for publishing
    ctx := context.Background()
    var node *p2p.Node
    if *bootstrap != "" {
        node, err = p2p.New(ctx, db, "/ip4/0.0.0.0/tcp/0", strings.Split(*bootstrap, ","))
        if err != nil { log.Fatal(err) }
        if err := node.Start(ctx); err != nil { log.Fatal(err) }
        defer node.Host.Close()
    }

    // Create website manifest
    manifest := &core.WebsiteManifest{
        Version:   "1.0",
        SitePub:   pub,
        Seq:       1,
        PrevCID:   "",
        TS:        core.NowTS(),
        MainFile:  *mainFile,
        Files:     make(map[string]string),
    }

    // Process all files in the website directory
    err = filepath.Walk(*websiteDir, func(path string, info os.FileInfo, err error) error {
        if err != nil { return err }
        if info.IsDir() { return nil }

        // Get relative path from website directory
        relPath, err := filepath.Rel(*websiteDir, path)
        if err != nil { return err }

        // Read file content
        content, err := os.ReadFile(path)
        if err != nil { return err }

        // Generate content CID
        contentCID := core.CIDForContent(content)
        
        // Store content
        if err := db.PutContent(contentCID, content); err != nil {
            return fmt.Errorf("failed to store content for %s: %v", relPath, err)
        }

        // Create file record
        fileRecord := &core.FileRecord{
            Version:    "1.0",
            SitePub:    pub,
            Path:       relPath,
            ContentCID: contentCID,
            MimeType:   core.GetMimeType(relPath),
            TS:         core.NowTS(),
        }

        // Generate ephemeral key for this file
        fileUpdatePub, fileUpdatePriv, err := bncrypto.GenerateSiteKey()
        if err != nil { return err }
        fileRecord.UpdatePub = fileUpdatePub

        // Sign file record
        fileRecordData, err := core.CanonicalMarshalFileRecordNoUpdateSig(fileRecord)
        if err != nil { return err }
        fileRecord.UpdateSig = ed25519.Sign(fileUpdatePriv, fileRecordData)

        // Store file record
        fileRecordCID := core.CIDForBytes(fileRecordData)
        if err := db.PutRecord(fileRecordCID, fileRecordData); err != nil {
            return fmt.Errorf("failed to store file record for %s: %v", relPath, err)
        }

        // Add to manifest
        manifest.Files[relPath] = contentCID

        fmt.Printf("Processed file: %s (CID: %s)\n", relPath, contentCID)
        return nil
    })

    if err != nil { log.Fatal(err) }

    // Generate ephemeral key for manifest
    manifestUpdatePub, manifestUpdatePriv, err := bncrypto.GenerateSiteKey()
    if err != nil { log.Fatal(err) }
    manifest.UpdatePub = manifestUpdatePub

    // Sign manifest
    manifestData, err := core.CanonicalMarshalWebsiteManifestNoUpdateSig(manifest)
    if err != nil { log.Fatal(err) }
    manifest.UpdateSig = ed25519.Sign(manifestUpdatePriv, manifestData)

    // Store manifest
    manifestCID := core.CIDForBytes(manifestData)
    if err := db.PutRecord(manifestCID, manifestData); err != nil {
        log.Fatal("failed to store manifest record")
    }

    // Store website manifest in store
    siteID := core.SiteIDFromPub(pub)
    if err := db.PutWebsiteManifest(siteID, manifestCID, manifestData); err != nil {
        log.Fatal("failed to store website manifest")
    }

    fmt.Printf("\nWebsite published successfully!\n")
    fmt.Printf("Site ID: %s\n", siteID)
    fmt.Printf("Main file: %s\n", *mainFile)
    fmt.Printf("Total files: %d\n", len(manifest.Files))
    fmt.Printf("Manifest CID: %s\n", manifestCID)
}

// addFile adds a single file to an existing website
func addFile() {
    fs := flag.NewFlagSet("add-file", flag.ExitOnError)
    keyPath := fs.String("key", "", "path to base64 ed25519 private key")
    data := fs.String("data", "./data", "data directory")
    filePath := fs.String("path", "", "file path within website (e.g., styles/main.css)")
    contentPath := fs.String("content", "", "path to file content")
    _ = fs.Parse(os.Args[2:])

    if *keyPath == "" || *filePath == "" || *contentPath == "" {
        log.Fatal("key, path, and content are required")
    }

    // Load private key
    keyData, err := os.ReadFile(*keyPath)
    if err != nil { log.Fatal(err) }
    privBytes, err := base64.StdEncoding.DecodeString(string(keyData))
    if err != nil { log.Fatal(err) }
    priv := ed25519.PrivateKey(privBytes)
    pub := priv.Public().(ed25519.PublicKey)

    // Open database
    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    // Read file content
    content, err := os.ReadFile(*contentPath)
    if err != nil { log.Fatal(err) }

    // Generate content CID
    contentCID := core.CIDForContent(content)
    
    // Store content
    if err := db.PutContent(contentCID, content); err != nil {
        log.Fatal("failed to store content")
    }

    // Create file record
    fileRecord := &core.FileRecord{
        Version:    "1.0",
        SitePub:    pub,
        Path:       *filePath,
        ContentCID: contentCID,
        MimeType:   core.GetMimeType(*filePath),
        TS:         core.NowTS(),
    }

    // Generate ephemeral key for this file
    fileUpdatePub, fileUpdatePriv, err := bncrypto.GenerateSiteKey()
    if err != nil { log.Fatal(err) }
    fileRecord.UpdatePub = fileUpdatePub

    // Sign file record
    fileRecordData, err := core.CanonicalMarshalFileRecordNoUpdateSig(fileRecord)
    if err != nil { log.Fatal(err) }
    fileRecord.UpdateSig = ed25519.Sign(fileUpdatePriv, fileRecordData)

    // Store file record
    fileRecordCID := core.CIDForBytes(fileRecordData)
    if err := db.PutRecord(fileRecordCID, fileRecordData); err != nil {
        log.Fatal("failed to store file record")
    }

    // Store file record in store
    siteID := core.SiteIDFromPub(pub)
    if err := db.PutFileRecord(siteID, *filePath, fileRecordCID, fileRecordData); err != nil {
        log.Fatal("failed to store file record")
    }

    fmt.Printf("File added successfully!\n")
    fmt.Printf("Site ID: %s\n", siteID)
    fmt.Printf("File path: %s\n", *filePath)
    fmt.Printf("Content CID: %s\n", contentCID)
    fmt.Printf("Record CID: %s\n", fileRecordCID)
}

// listWebsite lists all files in a website
func listWebsite() {
    fs := flag.NewFlagSet("list-website", flag.ExitOnError)
    data := fs.String("data", "./data", "data directory")
    siteID := fs.String("site", "", "site ID to list")
    _ = fs.Parse(os.Args[2:])

    if *siteID == "" {
        log.Fatal("site is required")
    }

    // Open database
    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    // Check if it's a multi-file website
    if db.HasWebsiteManifest(*siteID) {
        // Get website info
        info, err := db.GetWebsiteInfo(*siteID)
        if err != nil { log.Fatal(err) }

        fmt.Printf("Website: %s\n", *siteID)
        fmt.Printf("Main file: %s\n", info.MainFile)
        fmt.Printf("Total files: %d\n", info.FileCount)
        fmt.Printf("Last updated: %s\n", info.LastUpdated.Format(time.RFC3339))
        fmt.Printf("\nFiles:\n")
        
        for path, fileInfo := range info.Files {
            fmt.Printf("  %s (%s, %d bytes, %s)\n", 
                path, fileInfo.MimeType, fileInfo.Size, 
                fileInfo.LastUpdated.Format(time.RFC3339))
        }
    } else {
        // Check if it's a traditional single-file site
        hasHead, err := db.HasHead(*siteID)
        if err != nil { log.Fatal(err) }
        
        if hasHead {
            seq, headCID, err := db.GetHead(*siteID)
            if err != nil { log.Fatal(err) }
            
            fmt.Printf("Single-file site: %s\n", *siteID)
            fmt.Printf("Sequence: %d\n", seq)
            fmt.Printf("Head CID: %s\n", headCID)
        } else {
            fmt.Printf("Site %s not found\n", *siteID)
        }
    }
}

// getWebsiteInfo gets detailed information about a website
func getWebsiteInfo() {
    fs := flag.NewFlagSet("get-website-info", flag.ExitOnError)
    data := fs.String("data", "./data", "data directory")
    siteID := fs.String("site", "", "site ID to get info for")
    _ = fs.Parse(os.Args[2:])

    if *siteID == "" {
        log.Fatal("site is required")
    }

    // Open database
    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    // Check if it's a multi-file website
    if db.HasWebsiteManifest(*siteID) {
        // Get website info
        info, err := db.GetWebsiteInfo(*siteID)
        if err != nil { log.Fatal(err) }

        // Convert to JSON for pretty printing
        jsonData, err := json.MarshalIndent(info, "", "  ")
        if err != nil { log.Fatal(err) }

        fmt.Printf("Website Information:\n%s\n", string(jsonData))
    } else {
        fmt.Printf("Site %s is not a multi-file website\n", *siteID)
    }
}


