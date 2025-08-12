package main

import (
    "context"
    "crypto/ed25519"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "betanet/internal/core"
    "betanet/internal/p2p"
    "betanet/internal/store"
    "betanet/internal/wallet"

    fyne "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"
)

type nodeState struct {
    mu       sync.Mutex
    ctx      context.Context
    cancel   context.CancelFunc
    db       *store.Store
    node     *p2p.Node
    running  bool
}

func (s *nodeState) startNode(logArea *widget.Entry, data, listen, bootstrap string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    if s.running {
        return nil
    }
    db, err := store.Open(data)
    if err != nil { return err }
    ctx, cancel := context.WithCancel(context.Background())
    node, err := p2p.New(ctx, db, listen, splitCSV(bootstrap))
    if err != nil { cancel(); db.Close(); return err }
    if err := node.Start(ctx); err != nil { cancel(); db.Close(); return err }
    s.db = db
    s.ctx = ctx
    s.cancel = cancel
    s.node = node
    s.running = true
    appendLog(logArea, fmt.Sprintf("Node started on %s", listen))
    return nil
}

func (s *nodeState) stopNode(logArea *widget.Entry) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if !s.running { return }
    if s.cancel != nil { s.cancel() }
    if s.db != nil { _ = s.db.Close() }
    s.ctx = nil
    s.cancel = nil
    s.db = nil
    s.node = nil
    s.running = false
    appendLog(logArea, "Node stopped")
}

func appendLog(area *widget.Entry, line string) {
    area.SetText(area.Text + time.Now().Format("15:04:05 ") + line + "\n")
    area.CursorRow = len(strings.Split(area.Text, "\n"))
}

func splitCSV(s string) []string {
    if strings.TrimSpace(s) == "" { return nil }
    parts := strings.Split(s, ",")
    out := make([]string, 0, len(parts))
    for _, p := range parts { if strings.TrimSpace(p) != "" { out = append(out, strings.TrimSpace(p)) } }
    return out
}

func main() {
    tabFlag := flag.String("tab", "", "start on tab: node|wallet")
    flag.Parse()

    a := app.New()
    w := a.NewWindow("Betanet GUI")
    w.Resize(fyne.NewSize(900, 600))

    nodeTab := buildNodeTab(w)
    walletTab := buildWalletTab(w)

    tabs := container.NewAppTabs(
        container.NewTabItem("Node", nodeTab),
        container.NewTabItem("Wallet", walletTab),
    )
    if strings.EqualFold(*tabFlag, "wallet") {
        tabs.SelectIndex(1)
    } else {
        tabs.SelectIndex(0)
    }
    tabs.SetTabLocation(container.TabLocationTop)
    w.SetContent(tabs)
    w.ShowAndRun()
}

func buildNodeTab(win fyne.Window) fyne.CanvasObject {
    state := &nodeState{}

    dataEntry := widget.NewEntry()
    dataEntry.SetPlaceHolder("/home/user/.betanet/node")
    listenEntry := widget.NewEntry()
    listenEntry.SetText("/ip4/0.0.0.0/tcp/4001")
    bootstrapEntry := widget.NewEntry()

    logArea := widget.NewMultiLineEntry()
    logArea.SetMinRowsVisible(12)
    logArea.Disable()
    logScroll := container.NewScroll(logArea)
    logScroll.SetMinSize(fyne.NewSize(400, 300))

    pickDataBtn := widget.NewButton("Select Data Dir", func() {
        dlg := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
            if err == nil && uri != nil { dataEntry.SetText(uri.Path()) }
        }, win)
        dlg.Show()
    })

    startBtn := widget.NewButton("Start Node", func() {
        data := dataEntry.Text
        if data == "" { data = filepath.Join(os.Getenv("HOME"), ".betanet", "node") }
        if err := os.MkdirAll(data, 0o755); err != nil { dialog.ShowError(err, win); return }
        if err := state.startNode(logArea, data, listenEntry.Text, bootstrapEntry.Text); err != nil {
            dialog.ShowError(err, win)
        }
    })
    stopBtn := widget.NewButton("Stop Node", func() { state.stopNode(logArea) })

    grid := container.NewGridWithColumns(2,
        container.NewVBox(widget.NewLabel("Data Dir"), dataEntry, pickDataBtn),
        container.NewVBox(widget.NewLabel("Listen"), listenEntry, widget.NewLabel("Bootstrap (comma separated)"), bootstrapEntry),
    )

    return container.NewBorder(grid, container.NewHBox(startBtn, stopBtn), nil, nil, container.NewVSplit(logScroll, widget.NewRichTextWithText("")))
}

func buildWalletTab(win fyne.Window) fyne.CanvasObject {
    walletPath := widget.NewEntry()
    walletPath.SetPlaceHolder("/home/user/.betanet/wallet.json")
    mnemonic := widget.NewPasswordEntry()
    mnemonic.SetPlaceHolder("enter or generate a mnemonic")
    labelEntry := widget.NewEntry()
    labelEntry.SetPlaceHolder("site label (e.g., mysite)")
    nodeData := widget.NewEntry()
    nodeData.SetPlaceHolder("/home/user/.betanet/node")
    listen := widget.NewEntry()
    listen.SetText("/ip4/0.0.0.0/tcp/0")
    bootstrap := widget.NewEntry()
    contentPath := widget.NewEntry()
    contentPass := widget.NewPasswordEntry()
    contentPass.SetPlaceHolder("optional encryption passphrase")

    // Head/Deletion/Fetch UI
    headOut := widget.NewMultiLineEntry()
    headOut.Disable()
    headScroll := container.NewScroll(headOut)
    headScroll.SetMinSize(fyne.NewSize(300, 100))
    delRec := widget.NewEntry()
    delRec.SetPlaceHolder("record CID or prefix to delete (optional)")
    delCont := widget.NewEntry()
    delCont.SetPlaceHolder("content CID or prefix to delete (optional)")
    fetchCID := widget.NewEntry()
    fetchCID.SetPlaceHolder("content CID or prefix to fetch")
    fetchDecPass := widget.NewPasswordEntry()
    fetchDecPass.SetPlaceHolder("optional decrypt passphrase for fetched content")
    fetchOutPath := widget.NewEntry()
    fetchOutPath.SetPlaceHolder("/path/to/output")

    sitesList := widget.NewMultiLineEntry()
    sitesList.Disable()
    sitesScroll := container.NewScroll(sitesList)
    sitesScroll.SetMinSize(fyne.NewSize(300, 100))

    pickWalletBtn := widget.NewButton("Select Wallet", func() {
        dlg := dialog.NewFileOpen(func(uri fyne.URIReadCloser, err error) {
            if err == nil && uri != nil { walletPath.SetText(uri.URI().Path()) }
        }, win)
        dlg.Show()
    })
    newWalletBtn := widget.NewButton("New Wallet", func() {
        path := walletPath.Text
        if path == "" { path = filepath.Join(os.Getenv("HOME"), ".betanet", "wallet.json") }
        if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { dialog.ShowError(err, win); return }
        mn, err := wallet.NewMnemonic()
        if err != nil { dialog.ShowError(err, win); return }
        w := wallet.New()
        enc, err := wallet.EncryptWallet(w, mn)
        if err != nil { dialog.ShowError(err, win); return }
        if err := wallet.Save(path, enc); err != nil { dialog.ShowError(err, win); return }
        mnemonic.SetText(mn)
        dialog.ShowInformation("Wallet Created", fmt.Sprintf("Saved at %s", path), win)
    })

    addSiteBtn := widget.NewButton("Add Site", func() {
        w, master, err := openWalletPath(walletPath.Text, mnemonic.Text)
        if err != nil { dialog.ShowError(err, win); return }
        meta, pub, _, err := w.EnsureSite(master, labelEntry.Text)
        if err != nil { dialog.ShowError(err, win); return }
        if err := saveWalletPath(walletPath.Text, w, mnemonic.Text); err != nil { dialog.ShowError(err, win); return }
        dialog.ShowInformation("Site Added", fmt.Sprintf("%s\nSiteID: %s\nPub: %s", meta.Label, meta.SiteID[:12], hex.EncodeToString(pub)), win)
    })

    listSitesBtn := widget.NewButton("List Sites", func() {
        w, _, err := openWalletPath(walletPath.Text, mnemonic.Text)
        if err != nil { dialog.ShowError(err, win); return }
        var b strings.Builder
        for _, m := range w.Sites {
            fmt.Fprintf(&b, "%s: siteID=%s seq=%d head=%s\n", m.Label, short(m.SiteID), m.Seq, short(m.HeadRecCID))
        }
        sitesList.SetText(b.String())
    })

    pickContentBtn := widget.NewButton("Select Content", func() {
        dlg := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
            if err == nil && r != nil { contentPath.SetText(r.URI().Path()) }
        }, win)
        dlg.Show()
    })

    showHeadBtn := widget.NewButton("Show Head", func() {
        wlt, master, err := openWalletPath(walletPath.Text, mnemonic.Text)
        if err != nil { dialog.ShowError(err, win); return }
        // Derive site keys (does not change wallet state significantly)
        _, pub, _, err := wlt.EnsureSite(master, labelEntry.Text)
        if err != nil { dialog.ShowError(err, win); return }
        siteID := core.SiteIDFromPub(pub)
        data := nodeData.Text
        if data == "" { data = filepath.Join(os.Getenv("HOME"), ".betanet", "node") }
        db, err := store.Open(data)
        if err != nil { dialog.ShowError(err, win); return }
        defer db.Close()
        if has, _ := db.HasHead(siteID); !has {
            headOut.SetText("no head")
            return
        }
        seq, headCID, err := db.GetHead(siteID)
        if err != nil { dialog.ShowError(err, win); return }
        headOut.SetText(fmt.Sprintf("siteID=%s\nseq=%d\nheadCID=%s", short(siteID), seq, short(headCID)))
    })

    publishBtn := widget.NewButton("Publish", func() {
        // read wallet
        w, master, err := openWalletPath(walletPath.Text, mnemonic.Text)
        if err != nil { dialog.ShowError(err, win); return }
        meta, pub, priv, err := w.EnsureSite(master, labelEntry.Text)
        if err != nil { dialog.ShowError(err, win); return }
        // read content
        cnt, err := ioutil.ReadFile(contentPath.Text)
        if err != nil { dialog.ShowError(err, win); return }
        if strings.TrimSpace(contentPass.Text) != "" {
            enc, err := wallet.EncryptContent(contentPass.Text, cnt)
            if err != nil { dialog.ShowError(err, win); return }
            cnt = enc
        }
        // ensure data dir
        data := nodeData.Text
        if data == "" { data = filepath.Join(os.Getenv("HOME"), ".betanet", "node") }
        if err := os.MkdirAll(data, 0o755); err != nil { dialog.ShowError(err, win); return }
        // start ephemeral node
        db, err := store.Open(data)
        if err != nil { dialog.ShowError(err, win); return }
        ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
        defer cancel()
        node, err := p2p.New(ctx, db, listen.Text, splitCSV(bootstrap.Text))
        if err != nil { db.Close(); dialog.ShowError(err, win); return }
        if err := node.Start(ctx); err != nil { db.Close(); dialog.ShowError(err, win); return }

        // compute seq/prev
        siteID := core.SiteIDFromPub(pub)
        var seq uint64 = 1
        prevCID := ""
        if has, _ := db.HasHead(siteID); has {
            s, headCID, err := db.GetHead(siteID)
            if err != nil { db.Close(); dialog.ShowError(err, win); return }
            seq = s + 1
            prevCID = headCID
        }

        env, recCID, err := node.BuildUpdate(ed25519.PrivateKey(priv), ed25519.PublicKey(pub), cnt, seq, prevCID)
        if err != nil { db.Close(); dialog.ShowError(err, win); return }
        if err := db.PutRecord(recCID, env.Record); err != nil { db.Close(); dialog.ShowError(err, win); return }
        if len(env.Content) > 0 { if err := db.PutContent(core.CIDForContent(env.Content), env.Content); err != nil { db.Close(); dialog.ShowError(err, win); return } }
        if err := db.SetHead(siteID, seq, recCID); err != nil { db.Close(); dialog.ShowError(err, win); return }
        if err := node.BroadcastUpdate(ctx, *env); err != nil { db.Close(); dialog.ShowError(err, win); return }

        meta.Seq = seq
        meta.HeadRecCID = recCID
        meta.ContentCID = core.CIDForContent(cnt)
        w.Sites[labelEntry.Text] = meta
        if err := saveWalletPath(walletPath.Text, w, mnemonic.Text); err != nil { db.Close(); dialog.ShowError(err, win); return }

        _ = db.Close()
        dialog.ShowInformation("Published", fmt.Sprintf("seq=%d\nrecCID=%s\ncontentCID=%s", seq, short(recCID), short(meta.ContentCID)), win)
    })

    exportKeyBtn := widget.NewButton("Export Site Key", func() {
        w, master, err := openWalletPath(walletPath.Text, mnemonic.Text)
        if err != nil { dialog.ShowError(err, win); return }
        _, _, priv, err := w.EnsureSite(master, labelEntry.Text)
        if err != nil { dialog.ShowError(err, win); return }
        dialog.ShowInformation("Site Private Key (base64)", base64.StdEncoding.EncodeToString(priv), win)
    })

    deleteBtn := widget.NewButton("Delete (record/content)", func() {
        if strings.TrimSpace(delRec.Text) == "" && strings.TrimSpace(delCont.Text) == "" {
            dialog.ShowInformation("Delete", "Provide a record CID and/or content CID", win)
            return
        }
        wlt, master, err := openWalletPath(walletPath.Text, mnemonic.Text)
        if err != nil { dialog.ShowError(err, win); return }
        _, pub, priv, err := wlt.EnsureSite(master, labelEntry.Text)
        if err != nil { dialog.ShowError(err, win); return }
        data := nodeData.Text
        if data == "" { data = filepath.Join(os.Getenv("HOME"), ".betanet", "node") }
        if err := os.MkdirAll(data, 0o755); err != nil { dialog.ShowError(err, win); return }
        db, err := store.Open(data)
        if err != nil { dialog.ShowError(err, win); return }
        defer db.Close()
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        node, err := p2p.New(ctx, db, listen.Text, splitCSV(bootstrap.Text))
        if err != nil { dialog.ShowError(err, win); return }
        if err := node.Start(ctx); err != nil { dialog.ShowError(err, win); return }
        del := core.DeleteRecord{Version: "v1", SitePub: pub, TargetRec: strings.TrimSpace(delRec.Text), TargetCont: strings.TrimSpace(delCont.Text), TS: time.Now().Unix()}
        pre := walletPreimageDelete(del.SitePub, del.TargetRec, del.TargetCont, del.TS)
        del.Sig = ed25519.Sign(ed25519.PrivateKey(priv), pre)
        if err := node.BroadcastDelete(ctx, del); err != nil { dialog.ShowError(err, win); return }
        dialog.ShowInformation("Delete", "Delete broadcasted", win)
    })

    pickFetchOut := widget.NewButton("Select Output", func() {
        dlg := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
            if err == nil && wc != nil {
                fetchOutPath.SetText(wc.URI().Path())
                _ = wc.Close()
            }
        }, win)
        dlg.Show()
    })
    fetchBtn := widget.NewButton("Fetch Content", func() {
        if strings.TrimSpace(fetchCID.Text) == "" { dialog.ShowInformation("Fetch", "Provide a content CID or prefix", win); return }
        data := nodeData.Text
        if data == "" { data = filepath.Join(os.Getenv("HOME"), ".betanet", "node") }
        db, err := store.Open(data)
        if err != nil { dialog.ShowError(err, win); return }
        defer db.Close()
        cid := strings.TrimSpace(fetchCID.Text)
        if len(cid) < 64 {
            if full, err := db.ResolveContentCID(cid); err == nil { cid = full }
        }
        b, err := db.GetContent(cid)
        if err != nil { dialog.ShowError(err, win); return }
        if strings.TrimSpace(fetchDecPass.Text) != "" {
            pt, err := wallet.DecryptContent(fetchDecPass.Text, b)
            if err != nil { dialog.ShowError(err, win); return }
            b = pt
        }
        if strings.TrimSpace(fetchOutPath.Text) == "" {
            dialog.ShowInformation("Fetch", fmt.Sprintf("Fetched %d bytes", len(b)), win)
            return
        }
        if err := os.WriteFile(fetchOutPath.Text, b, 0644); err != nil { dialog.ShowError(err, win); return }
        dialog.ShowInformation("Fetch", fmt.Sprintf("Wrote %s (%d bytes)", fetchOutPath.Text, len(b)), win)
    })

    grid := container.NewVBox(
        widget.NewLabel("Wallet File"), container.NewBorder(nil, nil, nil, pickWalletBtn, walletPath),
        widget.NewLabel("Mnemonic"), mnemonic,
        container.NewGridWithColumns(2,
            container.NewVBox(widget.NewLabel("Site Label"), labelEntry, addSiteBtn),
            container.NewVBox(widget.NewLabel("Sites"), sitesScroll, listSitesBtn),
        ),
        widget.NewSeparator(),
        widget.NewLabel("Publish"),
        container.NewGridWithColumns(2,
            container.NewVBox(widget.NewLabel("Node Data Dir"), nodeData),
            container.NewVBox(widget.NewLabel("Bootstrap"), bootstrap),
        ),
        container.NewGridWithColumns(2,
            container.NewVBox(widget.NewLabel("Listen"), listen),
            container.NewVBox(widget.NewLabel("Content"), container.NewBorder(nil, nil, nil, pickContentBtn, contentPath), contentPass),
        ),
        container.NewHBox(publishBtn, exportKeyBtn, newWalletBtn),
        widget.NewSeparator(),
        widget.NewLabel("Head / Delete / Fetch"),
        container.NewHBox(showHeadBtn),
        headScroll,
        container.NewGridWithColumns(2,
            container.NewVBox(widget.NewLabel("Delete Record CID"), delRec),
            container.NewVBox(widget.NewLabel("Delete Content CID"), delCont),
        ),
        container.NewHBox(deleteBtn),
        container.NewGridWithColumns(2,
            container.NewVBox(widget.NewLabel("Fetch Content CID"), fetchCID, fetchDecPass),
            container.NewVBox(widget.NewLabel("Output Path"), container.NewBorder(nil, nil, nil, pickFetchOut, fetchOutPath)),
        ),
        container.NewHBox(fetchBtn),
    )

    return container.NewScroll(grid)
}

// walletPreimageDelete mirrors bncrypto.PreimageDelete without importing internal/crypto here
func walletPreimageDelete(sitePub []byte, targetRecCID, targetContentCID string, ts int64) []byte {
    // Reuse internal derivation by going through wallet->master when possible;
    // for the GUI, just compute the tagged SHA-256 like the node.
    h := sha256New()
    h.Write([]byte("bn-del-v1"))
    h.Write(sitePub)
    h.Write([]byte(targetRecCID))
    h.Write([]byte(targetContentCID))
    var t [8]byte
    u := uint64(ts)
    for i := 0; i < 8; i++ { t[7-i] = byte(u >> (8 * i)) }
    h.Write(t[:])
    return h.Sum(nil)
}

func sha256New() hashWriter { return hashWriter{h: sha256.New()} }

type hashWriter struct{ h interface{ Write([]byte) (int, error); Sum([]byte) []byte } }

func (w hashWriter) Write(b []byte) { _, _ = w.h.Write(b) }
func (w hashWriter) Sum(_ []byte) []byte { return w.h.Sum(nil) }

func openWalletPath(path, mnemonic string) (*wallet.Wallet, []byte, error) {
    if path == "" { return nil, nil, fmt.Errorf("wallet path required") }
    enc, err := wallet.Load(path)
    if err != nil { return nil, nil, err }
    w, err := wallet.DecryptWallet(enc, mnemonic)
    if err != nil { return nil, nil, err }
    master, err := wallet.MasterKeyFromMnemonic(mnemonic)
    if err != nil { return nil, nil, err }
    return w, master, nil
}

func saveWalletPath(path string, w *wallet.Wallet, mnemonic string) error {
    enc, err := wallet.EncryptWallet(w, mnemonic)
    if err != nil { return err }
    return wallet.Save(path, enc)
}

func short(s string) string {
    if len(s) > 12 { return s[:12] }
    return s
}


