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
    "betanet/internal/p2p"
    "betanet/internal/store"
    "betanet/internal/wallet"
)

func main() {
    log.SetFlags(0)
    if len(os.Args) < 2 {
        usage()
        return
    }
    switch os.Args[1] {
    case "new":
        cmdNew()
    case "add-site":
        cmdAddSite()
    case "list":
        cmdList()
    case "publish":
        cmdPublish()
    case "export-key":
        cmdExportKey()
    case "register-domain":
        cmdRegisterDomain()
    case "list-domains":
        cmdListDomains()
    case "resolve-domain":
        cmdResolveDomain()
    default:
        usage()
    }
}

func usage() {
    fmt.Println("betanet-wallet commands:")
    fmt.Println("  new -out /path/wallet.json")
    fmt.Println("  add-site -wallet /path/wallet.json -mnemonic \"...\" -label mysite")
    fmt.Println("  list -wallet /path/wallet.json -mnemonic \"...\"")
    fmt.Println("  publish -wallet /path/wallet.json -mnemonic \"...\" -label mysite -content /path/file [-encrypt-pass \"phrase\"] [-listen ...] [-bootstrap ...] [-data /path/db]")
    fmt.Println("  export-key -wallet /path/wallet.json -mnemonic \"...\" -label mysite")
    fmt.Println("  register-domain -wallet /path/wallet.json -mnemonic \"...\" -label mysite -domain mydomain.bn [-data /path/db]")
    fmt.Println("  list-domains -data /path/db")
    fmt.Println("  resolve-domain -data /path/db -domain mydomain.bn")
}

func cmdNew() {
    fs := flag.NewFlagSet("new", flag.ExitOnError)
    out := fs.String("out", "wallet.json", "wallet file path")
    _ = fs.Parse(os.Args[2:])

    mn, err := wallet.NewMnemonic()
    if err != nil { log.Fatal(err) }

    w := wallet.New()
    bytes, err := wallet.EncryptWallet(w, mn)
    if err != nil { log.Fatal(err) }
    if err := wallet.Save(*out, bytes); err != nil { log.Fatal(err) }

    fmt.Println("Created wallet:")
    fmt.Printf("  File: %s\n", *out)
    fmt.Println("  Mnemonic (STORE SAFELY, required to unlock):")
    fmt.Println(mn)
}

func openWallet(path, mnemonic string) (*wallet.Wallet, []byte) {
    enc, err := wallet.Load(path)
    if err != nil { log.Fatal(err) }
    w, err := wallet.DecryptWallet(enc, mnemonic)
    if err != nil { log.Fatal(err) }
    master, err := wallet.MasterKeyFromMnemonic(mnemonic)
    if err != nil { log.Fatal(err) }
    return w, master
}

func saveWallet(path string, w *wallet.Wallet, mnemonic string) {
    enc, err := wallet.EncryptWallet(w, mnemonic)
    if err != nil { log.Fatal(err) }
    if err := wallet.Save(path, enc); err != nil { log.Fatal(err) }
}

func cmdAddSite() {
    fs := flag.NewFlagSet("add-site", flag.ExitOnError)
    wf := fs.String("wallet", "wallet.json", "wallet file")
    mn := fs.String("mnemonic", "", "mnemonic (required)")
    label := fs.String("label", "", "site label")
    _ = fs.Parse(os.Args[2:])
    if *mn == "" || *label == "" { log.Fatal("missing -mnemonic or -label") }

    w, master := openWallet(*wf, *mn)
    meta, pub, _, err := w.EnsureSite(master, *label)
    if err != nil { log.Fatal(err) }
    saveWallet(*wf, w, *mn)

    fmt.Printf("Added site:\n  label=%s\n  siteID=%s\n  sitePub=%s\n",
        meta.Label, meta.SiteID[:12], hex.EncodeToString(pub))
}

func cmdList() {
    fs := flag.NewFlagSet("list", flag.ExitOnError)
    wf := fs.String("wallet", "wallet.json", "wallet file")
    mn := fs.String("mnemonic", "", "mnemonic (required)")
    _ = fs.Parse(os.Args[2:])
    if *mn == "" { log.Fatal("missing -mnemonic") }

    w, _ := openWallet(*wf, *mn)
    for _, m := range w.Sites {
        fmt.Printf("- %s: siteID=%s seq=%d head=%s\n",
            m.Label, m.SiteID[:12], m.Seq, short(m.HeadRecCID))
    }
}

func cmdPublish() {
    fs := flag.NewFlagSet("publish", flag.ExitOnError)
    wf := fs.String("wallet", "wallet.json", "wallet file")
    mn := fs.String("mnemonic", "", "mnemonic (required)")
    label := fs.String("label", "", "site label")
    content := fs.String("content", "", "file to publish")
    encPass := fs.String("encrypt-pass", "", "optional passphrase to encrypt content")
    data := fs.String("data", "./data", "node data dir")
    listen := fs.String("listen", "/ip4/0.0.0.0/tcp/0", "listen addr")
    bootstrap := fs.String("bootstrap", "", "comma-separated peers")
    _ = fs.Parse(os.Args[2:])
    if *mn == "" || *label == "" || *content == "" {
        log.Fatal("missing -mnemonic or -label or -content")
    }

    w, master := openWallet(*wf, *mn)
    meta, pub, priv, err := w.EnsureSite(master, *label)
    if err != nil { log.Fatal(err) }

    cnt, err := os.ReadFile(*content)
    if err != nil { log.Fatal(err) }
    if *encPass != "" {
        enc, err := wallet.EncryptContent(*encPass, cnt)
        if err != nil { log.Fatal(err) }
        cnt = enc
    }

    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

    meta.Seq = seq
    meta.HeadRecCID = recCID
    meta.ContentCID = core.CIDForContent(cnt)
    w.Sites[*label] = meta
    saveWallet(*wf, w, *mn)

    if err := node.BroadcastUpdate(ctx, *env); err != nil { log.Fatal(err) }

    fmt.Printf("Published %s seq=%d recCID=%s contentCID=%s\n",
        *label, seq, short(meta.HeadRecCID), short(meta.ContentCID))
}

func cmdExportKey() {
    fs := flag.NewFlagSet("export-key", flag.ExitOnError)
    wf := fs.String("wallet", "wallet.json", "wallet file")
    mn := fs.String("mnemonic", "", "mnemonic (required)")
    label := fs.String("label", "", "site label")
    _ = fs.Parse(os.Args[2:])
    if *mn == "" || *label == "" { log.Fatal("missing -mnemonic or -label") }

    w, master := openWallet(*wf, *mn)
    _, _, priv, err := w.EnsureSite(master, *label)
    if err != nil { log.Fatal(err) }
    fmt.Printf("ed25519 site private key (base64): %s\n",
        base64.StdEncoding.EncodeToString(priv))
}

func short(s string) string {
    if len(s) > 12 { return s[:12] }
    return s
}

// Domain name system commands
func cmdRegisterDomain() {
    fs := flag.NewFlagSet("register-domain", flag.ExitOnError)
    wf := fs.String("wallet", "wallet.json", "wallet file")
    mn := fs.String("mnemonic", "", "mnemonic (required)")
    label := fs.String("label", "", "site label")
    domain := fs.String("domain", "", "domain name (e.g., mysite.bn)")
    data := fs.String("data", "./data", "node data dir")
    _ = fs.Parse(os.Args[2:])
    
    if *mn == "" || *label == "" || *domain == "" {
        log.Fatal("missing -mnemonic, -label, or -domain")
    }
    
    // Validate domain format
    if !isValidDomain(*domain) {
        log.Fatal("invalid domain format: must be alphanumerical.alphanumerical (e.g., mysite.bn)")
    }
    
    w, master := openWallet(*wf, *mn)
    meta, pub, _, err := w.EnsureSite(master, *label)
    if err != nil { log.Fatal(err) }
    
    // Open database to register domain
    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()
    
    // Register the domain
    if err := db.RegisterDomain(*domain, meta.SiteID, pub); err != nil {
        log.Fatal("domain registration failed: ", err)
    }
    
    fmt.Printf("Domain registered successfully:\n  Domain: %s\n  Site: %s\n  SiteID: %s\n",
        *domain, *label, meta.SiteID[:12])
}

func cmdListDomains() {
    fs := flag.NewFlagSet("list-domains", flag.ExitOnError)
    data := fs.String("data", "./data", "node data dir")
    _ = fs.Parse(os.Args[2:])
    
    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()
    
    domains, err := db.ListDomains()
    if err != nil { log.Fatal(err) }
    
    if len(domains) == 0 {
        fmt.Println("No domains registered")
        return
    }
    
    fmt.Println("Registered domains:")
    for _, domain := range domains {
        siteID, err := db.ResolveDomain(domain)
        if err != nil {
            fmt.Printf("  %s -> ERROR: %v\n", domain, err)
            continue
        }
        fmt.Printf("  %s -> %s\n", domain, siteID[:12])
    }
}

func cmdResolveDomain() {
    fs := flag.NewFlagSet("resolve-domain", flag.ExitOnError)
    data := fs.String("data", "./data", "node data dir")
    domain := fs.String("domain", "", "domain name to resolve")
    _ = fs.Parse(os.Args[2:])
    
    if *domain == "" {
        log.Fatal("missing -domain")
    }
    
    db, err := store.Open(*data)
    if err != nil { log.Fatal(err) }
    defer db.Close()
    
    siteID, err := db.ResolveDomain(*domain)
    if err != nil {
        log.Fatal("domain resolution failed: ", err)
    }
    
    fmt.Printf("Domain resolved:\n  %s -> %s\n", *domain, siteID)
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


