package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	case "publish-website":
		cmdPublishWebsite()
	case "add-website-file":
		cmdAddWebsiteFile()
	case "list-website":
		cmdListWebsite()
	case "get-website-info":
		cmdGetWebsiteInfo()
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
	fmt.Println("")
	fmt.Println("Multi-file Website Commands:")
	fmt.Println("  publish-website -wallet /path/wallet.json -mnemonic \"...\" -label mysite -dir /path/website [-main index.html] [-listen ...] [-bootstrap ...] [-data /path/db]")
	fmt.Println("  add-website-file -wallet /path/wallet.json -mnemonic \"...\" -label mysite -path <filepath> -content /path/file [-listen ...] [-bootstrap ...] [-data /path/db]")
	fmt.Println("  list-website -wallet /path/wallet.json -mnemonic \"...\" -label mysite [-data /path/db]")
	fmt.Println("  get-website-info -wallet /path/wallet.json -mnemonic \"...\" -label mysite [-data /path/db]")
}

func cmdNew() {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	out := fs.String("out", "wallet.json", "wallet file path")
	_ = fs.Parse(os.Args[2:])

	mn, err := wallet.NewMnemonic()
	if err != nil {
		log.Fatal(err)
	}

	w := wallet.New()
	bytes, err := wallet.EncryptWallet(w, mn)
	if err != nil {
		log.Fatal(err)
	}
	if err := wallet.Save(*out, bytes); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Created wallet:")
	fmt.Printf("  File: %s\n", *out)
	fmt.Println("  Mnemonic (STORE SAFELY, required to unlock):")
	fmt.Println(mn)
}

func openWallet(path, mnemonic string) (*wallet.Wallet, []byte) {
	enc, err := wallet.Load(path)
	if err != nil {
		log.Fatal(err)
	}
	w, err := wallet.DecryptWallet(enc, mnemonic)
	if err != nil {
		log.Fatal(err)
	}
	master, err := wallet.MasterKeyFromMnemonic(mnemonic)
	if err != nil {
		log.Fatal(err)
	}
	return w, master
}

func saveWallet(path string, w *wallet.Wallet, mnemonic string) {
	enc, err := wallet.EncryptWallet(w, mnemonic)
	if err != nil {
		log.Fatal(err)
	}
	if err := wallet.Save(path, enc); err != nil {
		log.Fatal(err)
	}
}

func cmdAddSite() {
	fs := flag.NewFlagSet("add-site", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	label := fs.String("label", "", "site label")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" || *label == "" {
		log.Fatal("mnemonic and label are required")
	}

	w, master := openWallet(*wf, *mn)
	meta, _, _, err := w.EnsureSite(master, *label)
	if err != nil {
		log.Fatal(err)
	}
	saveWallet(*wf, w, *mn)

	fmt.Printf("Added site '%s' with ID: %s\n", *label, meta.SiteID)
}

func cmdList() {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" {
		log.Fatal("mnemonic is required")
	}

	w, _ := openWallet(*wf, *mn)
	fmt.Println("Sites in wallet:")
	for _, site := range w.Sites {
		fmt.Printf("  %s: %s\n", site.Label, site.SiteID)
	}
}

func cmdPublish() {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	label := fs.String("label", "", "site label")
	content := fs.String("content", "", "content file path")
	encryptPass := fs.String("encrypt-pass", "", "encryption passphrase (optional)")
	listen := fs.String("listen", "/ip4/0.0.0.0/tcp/0", "libp2p listen multiaddr")
	bootstrap := fs.String("bootstrap", "", "comma-separated peer multiaddrs")
	data := fs.String("data", "./data", "data directory")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" || *label == "" || *content == "" {
		log.Fatal("mnemonic, label, and content are required")
	}

	w, master := openWallet(*wf, *mn)
	meta, pub, _, err := w.EnsureSite(master, *label)
	if err != nil {
		log.Fatal(err)
	}

	// Read content file
	contentBytes, err := os.ReadFile(*content)
	if err != nil {
		log.Fatal(err)
	}

	// Encrypt if passphrase provided
	if *encryptPass != "" {
		contentBytes, err = wallet.EncryptContent(*encryptPass, contentBytes)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Open database
	db, err := store.Open(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Start node for publishing
	ctx := context.Background()
	var node *p2p.Node
	if *bootstrap != "" {
		node, err = p2p.New(ctx, db, *listen, strings.Split(*bootstrap, ","), nil)
		if err != nil {
			log.Fatal(err)
		}
		if err := node.Start(ctx); err != nil {
			log.Fatal(err)
		}
		defer node.Host.Close()
	}

	// Create update record
	record := &core.UpdateRecord{
		Version:    "1.0",
		SitePub:    pub,
		Seq:        1,
		PrevCID:    "",
		ContentCID: core.CIDForContent(contentBytes),
		TS:         core.NowTS(),
	}

	// Generate ephemeral key for this update
	updatePub, updatePriv, err := wallet.DeriveSiteKey(master, *label+"-update")
	if err != nil {
		log.Fatal(err)
	}
	record.UpdatePub = updatePub

	// Sign record
	recordData, err := core.CanonicalMarshalNoUpdateSig(record)
	if err != nil {
		log.Fatal(err)
	}
	record.UpdateSig = ed25519.Sign(updatePriv, recordData)

	// Store content and record
	if err := db.PutContent(record.ContentCID, contentBytes); err != nil {
		log.Fatal("failed to store content")
	}

	recordCID := core.CIDForBytes(recordData)
	if err := db.PutRecord(recordCID, recordData); err != nil {
		log.Fatal("failed to store record")
	}

	// Update site head
	if err := db.PutHead(meta.SiteID, record.Seq, recordCID); err != nil {
		log.Fatal("failed to update site head")
	}

	fmt.Printf("Published content for site '%s'\n", *label)
	fmt.Printf("Site ID: %s\n", meta.SiteID)
	fmt.Printf("Content CID: %s\n", record.ContentCID)
	fmt.Printf("Record CID: %s\n", recordCID)
}

func cmdExportKey() {
	fs := flag.NewFlagSet("export-key", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	label := fs.String("label", "", "site label")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" || *label == "" {
		log.Fatal("mnemonic and label are required")
	}

	w, master := openWallet(*wf, *mn)
	_, _, priv, err := w.EnsureSite(master, *label)
	if err != nil {
		log.Fatal(err)
	}

	keyData := base64.StdEncoding.EncodeToString(priv)
	fmt.Printf("Private key for site '%s':\n%s\n", *label, keyData)
}

func cmdRegisterDomain() {
	fs := flag.NewFlagSet("register-domain", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	label := fs.String("label", "", "site label")
	domain := fs.String("domain", "", "domain to register")
	data := fs.String("data", "./data", "data directory")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" || *label == "" || *domain == "" {
		log.Fatal("mnemonic, label, and domain are required")
	}

	w, master := openWallet(*wf, *mn)
	meta, _, _, err := w.EnsureSite(master, *label)
	if err != nil {
		log.Fatal(err)
	}

	// Open database
	db, err := store.Open(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Register domain
	if err := db.PutDomain(*domain, meta.SiteID); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Registered domain '%s' for site '%s'\n", *domain, *label)
	fmt.Printf("Site ID: %s\n", meta.SiteID)
}

func cmdListDomains() {
	fs := flag.NewFlagSet("list-domains", flag.ExitOnError)
	data := fs.String("data", "./data", "data directory")
	_ = fs.Parse(os.Args[2:])

	db, err := store.Open(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	domains, err := db.ListDomains()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Registered domains:")
	for domain, siteID := range domains {
		fmt.Printf("  %s -> %s\n", domain, siteID)
	}
}

func cmdResolveDomain() {
	fs := flag.NewFlagSet("resolve-domain", flag.ExitOnError)
	data := fs.String("data", "./data", "data directory")
	domain := fs.String("domain", "", "domain to resolve")
	_ = fs.Parse(os.Args[2:])

	if *domain == "" {
		log.Fatal("domain is required")
	}

	db, err := store.Open(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	siteID, err := db.ResolveDomain(*domain)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Domain '%s' resolves to site: %s\n", *domain, siteID)
}

// Multi-file website commands

func cmdPublishWebsite() {
	fs := flag.NewFlagSet("publish-website", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	label := fs.String("label", "", "site label")
	websiteDir := fs.String("dir", "", "path to website directory")
	mainFile := fs.String("main", "index.html", "main entry point file")
	data := fs.String("data", "./data", "data directory")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" || *label == "" || *websiteDir == "" {
		log.Fatal("mnemonic, label, and dir are required")
	}

	w, master := openWallet(*wf, *mn)
	meta, pub, _, err := w.EnsureSite(master, *label)
	if err != nil {
		log.Fatal(err)
	}

	// Check if main file exists
	mainFilePath := filepath.Join(*websiteDir, *mainFile)
	if _, err := os.Stat(mainFilePath); os.IsNotExist(err) {
		log.Fatalf("Main file %s does not exist", mainFilePath)
	}

	// Open database
	db, err := store.Open(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create website manifest
	manifest := &core.WebsiteManifest{
		Version:  "1.0",
		SitePub:  pub,
		Seq:      1,
		PrevCID:  "",
		TS:       core.NowTS(),
		MainFile: *mainFile,
		Files:    make(map[string]string),
	}

	// Process all files in the website directory
	err = filepath.Walk(*websiteDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Get relative path from website directory
		relPath, err := filepath.Rel(*websiteDir, path)
		if err != nil {
			return err
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

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
		fileUpdatePub, fileUpdatePriv, err := wallet.DeriveSiteKey(master, *label+"-file-"+relPath)
		if err != nil {
			return err
		}
		fileRecord.UpdatePub = fileUpdatePub

		// Sign file record
		fileRecordData, err := core.CanonicalMarshalFileRecordNoUpdateSig(fileRecord)
		if err != nil {
			return err
		}
		fileRecord.UpdateSig = ed25519.Sign(fileUpdatePriv, fileRecordData)

		// Store file record
		fileRecordCID := core.CIDForBytes(fileRecordData)
		if err := db.PutRecord(fileRecordCID, fileRecordData); err != nil {
			return fmt.Errorf("failed to store file record for %s: %v", relPath, err)
		}

		// Store file record in store
		if err := db.PutFileRecord(meta.SiteID, relPath, fileRecordCID, fileRecordData); err != nil {
			return fmt.Errorf("failed to store file record for %s: %v", relPath, err)
		}

		// Add to manifest
		manifest.Files[relPath] = contentCID

		fmt.Printf("Processed file: %s (CID: %s)\n", relPath, contentCID)
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// Generate ephemeral key for manifest
	manifestUpdatePub, manifestUpdatePriv, err := wallet.DeriveSiteKey(master, *label+"-manifest")
	if err != nil {
		log.Fatal(err)
	}
	manifest.UpdatePub = manifestUpdatePub

	// Sign manifest
	manifestData, err := core.CanonicalMarshalWebsiteManifestNoUpdateSig(manifest)
	if err != nil {
		log.Fatal(err)
	}
	manifest.UpdateSig = ed25519.Sign(manifestUpdatePriv, manifestData)

	// Store manifest
	manifestCID := core.CIDForBytes(manifestData)
	if err := db.PutRecord(manifestCID, manifestData); err != nil {
		log.Fatal("failed to store manifest record")
	}

	// Store website manifest in store
	if err := db.PutWebsiteManifest(meta.SiteID, manifestCID, manifestData); err != nil {
		log.Fatal("failed to store website manifest")
	}

	fmt.Printf("\nWebsite published successfully!\n")
	fmt.Printf("Site ID: %s\n", meta.SiteID)
	fmt.Printf("Main file: %s\n", *mainFile)
	fmt.Printf("Total files: %d\n", len(manifest.Files))
	fmt.Printf("Manifest CID: %s\n", manifestCID)
}

func cmdAddWebsiteFile() {
	fs := flag.NewFlagSet("add-website-file", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	label := fs.String("label", "", "site label")
	filePath := fs.String("path", "", "file path within website (e.g., styles/main.css)")
	contentPath := fs.String("content", "", "path to file content")
	data := fs.String("data", "./data", "data directory")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" || *label == "" || *filePath == "" || *contentPath == "" {
		log.Fatal("mnemonic, label, path, and content are required")
	}

	w, master := openWallet(*wf, *mn)
	meta, pub, _, err := w.EnsureSite(master, *label)
	if err != nil {
		log.Fatal(err)
	}

	// Read file content
	content, err := os.ReadFile(*contentPath)
	if err != nil {
		log.Fatal(err)
	}

	// Open database
	db, err := store.Open(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
	fileUpdatePub, fileUpdatePriv, err := wallet.DeriveSiteKey(master, *label+"-file-"+*filePath)
	if err != nil {
		log.Fatal(err)
	}
	fileRecord.UpdatePub = fileUpdatePub

	// Sign file record
	fileRecordData, err := core.CanonicalMarshalFileRecordNoUpdateSig(fileRecord)
	if err != nil {
		log.Fatal(err)
	}
	fileRecord.UpdateSig = ed25519.Sign(fileUpdatePriv, fileRecordData)

	// Store file record
	fileRecordCID := core.CIDForBytes(fileRecordData)
	if err := db.PutRecord(fileRecordCID, fileRecordData); err != nil {
		log.Fatal("failed to store file record")
	}

	// Store file record in store
	if err := db.PutFileRecord(meta.SiteID, *filePath, fileRecordCID, fileRecordData); err != nil {
		log.Fatal("failed to store file record")
	}

	fmt.Printf("File added successfully!\n")
	fmt.Printf("Site ID: %s\n", meta.SiteID)
	fmt.Printf("File path: %s\n", *filePath)
	fmt.Printf("Content CID: %s\n", contentCID)
	fmt.Printf("Record CID: %s\n", fileRecordCID)
}

func cmdListWebsite() {
	fs := flag.NewFlagSet("list-website", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	label := fs.String("label", "", "site label")
	data := fs.String("data", "./data", "data directory")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" || *label == "" {
		log.Fatal("mnemonic and label are required")
	}

	w, master := openWallet(*wf, *mn)
	meta, _, _, err := w.EnsureSite(master, *label)
	if err != nil {
		log.Fatal(err)
	}

	// Open database
	db, err := store.Open(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if it's a multi-file website
	if db.HasWebsiteManifest(meta.SiteID) {
		// Get website info
		info, err := db.GetWebsiteInfo(meta.SiteID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Website: %s (%s)\n", meta.Label, meta.SiteID)
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
		fmt.Printf("Site '%s' is not a multi-file website\n", meta.Label)
	}
}

func cmdGetWebsiteInfo() {
	fs := flag.NewFlagSet("get-website-info", flag.ExitOnError)
	wf := fs.String("wallet", "wallet.json", "wallet file")
	mn := fs.String("mnemonic", "", "mnemonic (required)")
	label := fs.String("label", "", "site label")
	data := fs.String("data", "./data", "data directory")
	_ = fs.Parse(os.Args[2:])

	if *mn == "" || *label == "" {
		log.Fatal("mnemonic and label are required")
	}

	w, master := openWallet(*wf, *mn)
	meta, _, _, err := w.EnsureSite(master, *label)
	if err != nil {
		log.Fatal(err)
	}

	// Open database
	db, err := store.Open(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if it's a multi-file website
	if db.HasWebsiteManifest(meta.SiteID) {
		// Get website info
		info, err := db.GetWebsiteInfo(meta.SiteID)
		if err != nil {
			log.Fatal(err)
		}

		// Convert to JSON for pretty printing
		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Website Information for '%s':\n%s\n", meta.Label, string(jsonData))
	} else {
		fmt.Printf("Site '%s' is not a multi-file website\n", meta.Label)
	}
}
