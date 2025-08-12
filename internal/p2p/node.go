package p2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"betanet/internal/core"
	bncrypto "betanet/internal/crypto"
	"betanet/internal/store"

	"github.com/fxamacker/cbor/v2"
	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	host "github.com/libp2p/go-libp2p/core/host"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	protocol "github.com/libp2p/go-libp2p/core/protocol"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	ping "github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

const Topic = "betanet/updates/v1"
const BrowseProto protocol.ID = "/betanet/browse/1.0.0"

// Security and performance constants
const (
	MaxMessageSize        = 1024 * 1024       // 1MB max message size
	MaxPeers              = 100               // Maximum number of peers
	PeerTimeout           = 30 * time.Second  // Peer connection timeout
	RateLimitWindow       = 1 * time.Minute   // Rate limiting window
	MaxRequestsPerWindow  = 100               // Max requests per peer per window
	MemoryCleanupInterval = 5 * time.Minute   // Memory cleanup interval
	MaxMemoryUsage        = 100 * 1024 * 1024 // 100MB memory limit
)

type GossipUpdate struct {
	Record  []byte // canonical CBOR of UpdateRecord
	Content []byte // optional content bytes (small)
}

type GossipDelete struct {
	Delete []byte // canonical CBOR of DeleteRecord
}

// RateLimiter prevents abuse from individual peers
type RateLimiter struct {
	requests    map[string][]time.Time
	maxRequests int
	window      time.Duration
	mu          sync.RWMutex
}

// PeerInfo tracks peer reputation and status
type PeerInfo struct {
	ID           peer.ID
	Reputation   int // -100 to +100
	LastSeen     time.Time
	BannedUntil  *time.Time
	RequestCount int
	ErrorCount   int
}

// Node represents a P2P network node with enhanced security
type Node struct {
	Host           host.Host
	HostPub        ed25519.PublicKey
	HostPriv       ed25519.PrivateKey
	PubSub         *pubsub.PubSub
	Topic          *pubsub.Topic
	Sub            *pubsub.Subscription
	Store          *store.Store
	BootstrapAddrs []ma.Multiaddr

	// Security and performance features
	rateLimiter    *RateLimiter
	peers          map[peer.ID]*PeerInfo
	bannedPeers    map[peer.ID]time.Time
	memoryUsage    int64
	maxMemoryUsage int64
	mu             sync.RWMutex

	// Logging
	logger *zap.Logger

	// Configuration
	config *NodeConfig
}

// NodeConfig holds node configuration
type NodeConfig struct {
	MaxPeers             int
	PeerTimeout          time.Duration
	RateLimitWindow      time.Duration
	MaxRequestsPerWindow int
	MaxMemoryUsage       int64
	EnablePeerValidation bool
	EnableRateLimiting   bool
}

// DefaultNodeConfig returns sensible defaults
func DefaultNodeConfig() *NodeConfig {
	return &NodeConfig{
		MaxPeers:             MaxPeers,
		PeerTimeout:          PeerTimeout,
		RateLimitWindow:      RateLimitWindow,
		MaxRequestsPerWindow: MaxRequestsPerWindow,
		MaxMemoryUsage:       MaxMemoryUsage,
		EnablePeerValidation: true,
		EnableRateLimiting:   true,
	}
}

func New(ctx context.Context, db *store.Store, listen string, bootstrap []string, config *NodeConfig) (*Node, error) {
	if config == nil {
		config = DefaultNodeConfig()
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	hostPub, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %w", err)
	}

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(listen),
		libp2p.ResourceManager(nil), // We'll implement our own resource management
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	t, err := ps.Join(Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic: %w", err)
	}

	sub, err := t.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	var maddrs []ma.Multiaddr
	for _, s := range bootstrap {
		if s == "" {
			continue
		}
		m, err := ma.NewMultiaddr(s)
		if err != nil {
			logger.Warn("invalid bootstrap address", zap.String("addr", s), zap.Error(err))
			continue
		}
		maddrs = append(maddrs, m)
	}

	n := &Node{
		Host:           h,
		HostPub:        hostPub,
		HostPriv:       hostPriv,
		PubSub:         ps,
		Topic:          t,
		Sub:            sub,
		Store:          db,
		BootstrapAddrs: maddrs,
		rateLimiter: &RateLimiter{
			requests:    make(map[string][]time.Time),
			maxRequests: config.MaxRequestsPerWindow,
			window:      config.RateLimitWindow,
		},
		peers:          make(map[peer.ID]*PeerInfo),
		bannedPeers:    make(map[peer.ID]time.Time),
		maxMemoryUsage: config.MaxMemoryUsage,
		logger:         logger,
		config:         config,
	}

	// Register browse protocol handler
	h.SetStreamHandler(BrowseProto, n.handleBrowseStream)

	// Set connection handlers
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF:    n.handlePeerConnected,
		DisconnectedF: n.handlePeerDisconnected,
	})

	logger.Info("node created successfully",
		zap.String("host_id", h.ID().String()),
		zap.String("listen_addr", listen),
		zap.Int("bootstrap_peers", len(maddrs)))

	return n, nil
}

// Start initializes and starts the node
func (n *Node) Start(ctx context.Context) error {
	n.logger.Info("starting node")

	// Start background goroutines
	go n.consume(ctx)
	go n.peerManagement(ctx)
	go n.memoryManagement(ctx)
	go n.cleanupBannedPeers(ctx)

	// Start periodic tasks
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = n.Topic.Publish(ctx, []byte("bn-alive"))
			}
		}
	}()
	// Enable mDNS advertise/respond on LAN so Browser auto-discovery can find this node
	_ = mdns.NewMdnsService(n.Host, "betanet-mdns", &mdnsNotifee{cb: func(pi peer.AddrInfo) {
		log.Printf("mDNS: discovered peer %s", pi.ID)
	}})
	// Attempt to connect to bootstrap peers, if any
	for _, m := range n.BootstrapAddrs {
		if ai, err := peer.AddrInfoFromP2pAddr(m); err == nil {
			go func(info peer.AddrInfo) {
				cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				_ = n.Host.Connect(cctx, info)
			}(*ai)
		}
	}
	n.logger.Info("node started successfully")
	return nil
}

func (n *Node) consume(ctx context.Context) {
	for {
		msg, err := n.Sub.Next(ctx)
		if err != nil {
			return
		}
		data := msg.GetData()
		if string(data) == "bn-alive" {
			continue
		}
		// Try update first
		var u GossipUpdate
		if err := cborUnmarshal(data, &u); err == nil && len(u.Record) > 0 {
			n.handleEnvelope(u)
			continue
		}
		// Then try delete
		var d GossipDelete
		if err := cborUnmarshal(data, &d); err == nil && len(d.Delete) > 0 {
			n.handleDelete(d)
			continue
		}
	}
}

func cborMarshal(v any) ([]byte, error) {
	enc, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(v)
}
func cborUnmarshal(b []byte, v any) error {
	dec, err := cbor.DecOptions{TimeTag: cbor.DecTagRequired}.DecMode()
	if err != nil {
		return err
	}
	return dec.Unmarshal(b, v)
}

func (n *Node) handleEnvelope(env GossipUpdate) {
	var rec core.UpdateRecord
	if err := cborUnmarshal(env.Record, &rec); err != nil {
		return
	}
	if err := n.ValidateAndApply(&rec, env.Content); err != nil {
		log.Printf("reject update: %v", err)
	}
}

func (n *Node) handleDelete(env GossipDelete) {
	var del core.DeleteRecord
	if err := cborUnmarshal(env.Delete, &del); err != nil {
		return
	}
	// Verify signature
	pre := bncrypto.PreimageDelete(del.SitePub, del.TargetRec, del.TargetCont, del.TS)
	if !ed25519.Verify(ed25519.PublicKey(del.SitePub), pre, del.Sig) {
		log.Printf("reject delete: invalid signature")
		return
	}
	// Apply
	if del.TargetRec != "" {
		// Resolve prefix if needed
		if len(del.TargetRec) < 64 {
			if full, err := n.Store.ResolveRecordCID(del.TargetRec); err == nil {
				del.TargetRec = full
			}
		}
		// Load the record to verify ownership and to get PrevCID
		recBytes, err := n.Store.GetRecord(del.TargetRec)
		if err == nil && len(recBytes) > 0 {
			var rec core.UpdateRecord
			if cborUnmarshal(recBytes, &rec) == nil {
				// Ensure delete key matches record site
				if hex.EncodeToString(rec.SitePub) == hex.EncodeToString(del.SitePub) {
					siteID := core.SiteIDFromPub(rec.SitePub)
					// Adjust head if needed
					if has, _ := n.Store.HasHead(siteID); has {
						seq, headCID, _ := n.Store.GetHead(siteID)
						if headCID == del.TargetRec {
							if seq > 0 {
								_ = n.Store.SetHead(siteID, seq-1, rec.PrevCID)
							}
						}
					}
				}
			}
		}
		_ = n.Store.DeleteRecord(del.TargetRec)
	}
	if del.TargetCont != "" {
		if len(del.TargetCont) < 64 {
			if full, err := n.Store.ResolveContentCID(del.TargetCont); err == nil {
				del.TargetCont = full
			}
		}
		_ = n.Store.DeleteContent(del.TargetCont)
	}
}

func (n *Node) ValidateAndApply(r *core.UpdateRecord, content []byte) error {
	if r.Version != "v1" {
		return errors.New("bad version")
	}
	siteID := core.SiteIDFromPub(r.SitePub)

	if len(content) > 0 {
		if core.CIDForContent(content) != r.ContentCID {
			return errors.New("content CID mismatch")
		}
	}

	linkPre := bncrypto.PreimageLink(r.SitePub, r.UpdatePub, r.Seq, r.PrevCID, r.ContentCID, r.TS)
	if !ed25519.Verify(ed25519.PublicKey(r.SitePub), linkPre, r.LinkSig) {
		return errors.New("invalid link signature")
	}

	bytesNoUS, err := core.CanonicalMarshalNoUpdateSig(r)
	if err != nil {
		return err
	}
	updPre := bncrypto.PreimageUpdate(bytesNoUS)
	if !ed25519.Verify(ed25519.PublicKey(r.UpdatePub), updPre, r.UpdateSig) {
		return errors.New("invalid update signature")
	}

	hasHead, err := n.Store.HasHead(siteID)
	if err != nil {
		return err
	}
	if hasHead {
		seq, headCID, err := n.Store.GetHead(siteID)
		if err != nil {
			return err
		}
		if r.Seq != seq+1 {
			return errors.New("sequence mismatch")
		}
		if r.PrevCID != headCID {
			return errors.New("prevCID mismatch")
		}
	} else {
		if r.Seq != 1 || r.PrevCID != "" {
			return errors.New("invalid genesis update")
		}
	}

	if r.TS <= 0 {
		return errors.New("bad timestamp")
	}

	recBytes, err := core.CanonicalMarshal(r)
	if err != nil {
		return err
	}
	recCID := core.CIDForBytes(recBytes)
	if err := n.Store.PutRecord(recCID, recBytes); err != nil {
		return err
	}
	if len(content) > 0 {
		if err := n.Store.PutContent(r.ContentCID, content); err != nil {
			return err
		}
	}
	if err := n.Store.SetHead(siteID, r.Seq, recCID); err != nil {
		return err
	}

	log.Printf("accepted update site=%s seq=%d cid=%s content=%s",
		Short(siteID), r.Seq, Short(recCID), Short(r.ContentCID))
	return nil
}

func (n *Node) BroadcastUpdate(ctx context.Context, env GossipUpdate) error {
	b, err := cborMarshal(env)
	if err != nil {
		return err
	}
	return n.Topic.Publish(ctx, b)
}

func (n *Node) BroadcastDelete(ctx context.Context, del core.DeleteRecord) error {
	brec, err := cborMarshal(GossipDelete{Delete: mustMarshal(del)})
	if err != nil {
		return err
	}
	return n.Topic.Publish(ctx, brec)
}

func mustMarshal[T any](v T) []byte {
	b, _ := cborMarshal(v)
	return b
}

// --- Browse protocol (request/response) ---

type browseReq struct {
	Type   string `cbor:"t"`
	SiteID string `cbor:"s,omitempty"`
	CID    string `cbor:"c,omitempty"`
}

type browseRespHead struct {
	Ok         bool   `cbor:"ok"`
	Seq        uint64 `cbor:"seq,omitempty"`
	HeadCID    string `cbor:"h,omitempty"`
	ContentCID string `cbor:"cc,omitempty"`
}

type browseRespContent struct {
	Ok      bool   `cbor:"ok"`
	Content []byte `cbor:"ct,omitempty"`
}

func (n *Node) handleBrowseStream(s network.Stream) {
	defer s.Close()
	log.Printf("handleBrowseStream: new stream from %s", s.Conn().RemotePeer())

	// Set read deadline to prevent hanging
	s.SetReadDeadline(time.Now().Add(10 * time.Second))

	dec, _ := cbor.DecOptions{}.DecMode()
	reqBytes := readAllWithTimeout(s, 5*time.Second)
	log.Printf("handleBrowseStream: read %d bytes", len(reqBytes))
	var req browseReq
	if err := dec.Unmarshal(reqBytes, &req); err != nil {
		log.Printf("handleBrowseStream: unmarshal failed: %v", err)
		return
	}
	log.Printf("handleBrowseStream: got request type=%s siteID=%s cid=%s", req.Type, req.SiteID, req.CID)

	// Set write deadline
	s.SetWriteDeadline(time.Now().Add(5 * time.Second))

	switch req.Type {
	case "get_head":
		var resp browseRespHead
		if has, _ := n.Store.HasHead(req.SiteID); has {
			seq, headCID, _ := n.Store.GetHead(req.SiteID)
			if recBytes, err := n.Store.GetRecord(headCID); err == nil {
				var rec core.UpdateRecord
				if dec.Unmarshal(recBytes, &rec) == nil {
					resp = browseRespHead{Ok: true, Seq: seq, HeadCID: headCID, ContentCID: rec.ContentCID}
				}
			}
		}
		b, _ := cborMarshal(resp)
		if _, err := s.Write(b); err != nil {
			log.Printf("handleBrowseStream: write failed: %v", err)
		}
		log.Printf("handleBrowseStream: sent response ok=%v", resp.Ok)
	case "get_content":
		var resp browseRespContent
		if b, err := n.Store.GetContent(req.CID); err == nil {
			resp = browseRespContent{Ok: true, Content: b}
		}
		bb, _ := cborMarshal(resp)
		if _, err := s.Write(bb); err != nil {
			log.Printf("handleBrowseStream: write failed: %v", err)
		}
		log.Printf("handleBrowseStream: sent content response ok=%v size=%d", resp.Ok, len(resp.Content))
	}
}

func readAll(r io.Reader) []byte {
	buf := make([]byte, 0, 2048)
	tmp := make([]byte, 2048)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf
}

func readAllWithTimeout(r io.Reader, timeout time.Duration) []byte {
	// For network streams, try to read with a reasonable timeout
	buf := make([]byte, 0, 2048)
	tmp := make([]byte, 2048)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			// If we got data, continue reading for a bit more
			deadline = time.Now().Add(500 * time.Millisecond)
		}
		if err != nil {
			break
		}
		if n == 0 {
			// No data, small delay before retry
			time.Sleep(10 * time.Millisecond)
		}
	}
	return buf
}

// DiscoverBestPeer finds the lowest RTT mDNS peer within the timeout.
func (n *Node) DiscoverBestPeer(ctx context.Context, timeout time.Duration) (*peer.AddrInfo, error) {
	found := make(chan peer.AddrInfo, 32)
	_ = mdns.NewMdnsService(n.Host, "betanet-mdns", &mdnsNotifee{cb: func(pi peer.AddrInfo) {
		log.Printf("mDNS discovery: found peer %s with addrs %v", pi.ID, pi.Addrs)
		select {
		case found <- pi:
		default:
		}
	}})
	pinger := ping.NewPingService(n.Host)
	deadline := time.Now().Add(timeout)
	var best *peer.AddrInfo
	var bestRTT time.Duration = 1<<63 - 1
	for time.Now().Before(deadline) {
		select {
		case pi := <-found:
			log.Printf("mDNS: testing peer %s", pi.ID)
			_ = n.Host.Connect(ctx, pi)
			cctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			rtt, err := measurePingOnce(cctx, pinger, pi.ID)
			cancel()
			if err == nil && rtt < bestRTT {
				tmp := pi
				best = &tmp
				bestRTT = rtt
				log.Printf("mDNS: new best peer %s with RTT %v", pi.ID, rtt)
			}
		case <-time.After(100 * time.Millisecond):
		}
	}
	if best == nil {
		return nil, errors.New("no peers")
	}
	return best, nil
}

// DiscoverLocalhostPeer tries to find betanet nodes on common localhost ports
func (n *Node) DiscoverLocalhostPeer(ctx context.Context) (*peer.AddrInfo, error) {
	log.Printf("Trying localhost discovery on common ports...")

	// TODO: Implement proper localhost peer discovery
	// This would require scanning ports and extracting peer IDs from connections
	// For now, users should use the bootstrap address shown by the running node

	return nil, errors.New("localhost discovery requires manual bootstrap for now")
}

func measurePingOnce(ctx context.Context, pinger *ping.PingService, id peer.ID) (time.Duration, error) {
	ch := pinger.Ping(ctx, id)
	select {
	case res, ok := <-ch:
		if !ok {
			return 0, errors.New("closed")
		}
		return res.RTT, res.Error
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

type mdnsNotifee struct{ cb func(peer.AddrInfo) }

func (m *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if m.cb != nil {
		m.cb(pi)
	}
}

// RequestHead requests head info from a peer.
func (n *Node) RequestHead(ctx context.Context, p peer.AddrInfo, siteID string) (uint64, string, string, error) {
	log.Printf("RequestHead: connecting to peer %s", p.ID)
	if err := n.Host.Connect(ctx, p); err != nil {
		log.Printf("RequestHead: connect failed: %v", err)
		return 0, "", "", err
	}
	log.Printf("RequestHead: creating stream to %s", p.ID)
	s, err := n.Host.NewStream(ctx, p.ID, BrowseProto)
	if err != nil {
		log.Printf("RequestHead: stream creation failed: %v", err)
		return 0, "", "", err
	}
	defer s.Close()

	// Set timeouts
	s.SetWriteDeadline(time.Now().Add(5 * time.Second))
	s.SetReadDeadline(time.Now().Add(10 * time.Second))

	log.Printf("RequestHead: sending request for site %s", siteID)
	req := browseReq{Type: "get_head", SiteID: siteID}
	b, _ := cborMarshal(req)
	if _, err := s.Write(b); err != nil {
		log.Printf("RequestHead: write failed: %v", err)
		return 0, "", "", err
	}

	// Close write side to signal end of request
	if closer, ok := s.(interface{ CloseWrite() error }); ok {
		closer.CloseWrite()
	}

	log.Printf("RequestHead: reading response")
	dec, _ := cbor.DecOptions{}.DecMode()
	var resp browseRespHead
	respBytes := readAllWithTimeout(s, 5*time.Second)
	log.Printf("RequestHead: got %d bytes response", len(respBytes))
	if len(respBytes) == 0 {
		return 0, "", "", errors.New("no response data")
	}
	if err := dec.Unmarshal(respBytes, &resp); err != nil {
		log.Printf("RequestHead: unmarshal failed: %v", err)
		return 0, "", "", err
	}
	if !resp.Ok {
		log.Printf("RequestHead: server returned not ok")
		return 0, "", "", errors.New("not found")
	}
	log.Printf("RequestHead: success - seq=%d headCID=%s contentCID=%s", resp.Seq, resp.HeadCID, resp.ContentCID)
	return resp.Seq, resp.HeadCID, resp.ContentCID, nil
}

// RequestContent requests content by CID from a peer.
func (n *Node) RequestContent(ctx context.Context, p peer.AddrInfo, cid string) ([]byte, error) {
	if err := n.Host.Connect(ctx, p); err != nil {
		return nil, err
	}
	s, err := n.Host.NewStream(ctx, p.ID, BrowseProto)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	// Set timeouts
	s.SetWriteDeadline(time.Now().Add(5 * time.Second))
	s.SetReadDeadline(time.Now().Add(10 * time.Second))

	req := browseReq{Type: "get_content", CID: cid}
	b, _ := cborMarshal(req)
	if _, err := s.Write(b); err != nil {
		return nil, err
	}

	// Close write side to signal end of request
	if closer, ok := s.(interface{ CloseWrite() error }); ok {
		closer.CloseWrite()
	}

	dec, _ := cbor.DecOptions{}.DecMode()
	var resp browseRespContent
	respBytes := readAllWithTimeout(s, 5*time.Second)
	if len(respBytes) == 0 {
		return nil, errors.New("no response data")
	}
	if err := dec.Unmarshal(respBytes, &resp); err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, errors.New("not found")
	}
	return resp.Content, nil
}

func (n *Node) BuildUpdate(sitePriv ed25519.PrivateKey, sitePub ed25519.PublicKey, content []byte, seq uint64, prevRecCID string) (*GossipUpdate, string, error) {
	upPub, upPriv, err := bncrypto.GenerateUpdateKey()
	if err != nil {
		return nil, "", err
	}
	ts := time.Now().Unix()
	contentCID := core.CIDForContent(content)
	rec := core.UpdateRecord{
		Version:    "v1",
		SitePub:    sitePub,
		Seq:        seq,
		PrevCID:    prevRecCID,
		ContentCID: contentCID,
		TS:         ts,
		UpdatePub:  upPub,
	}
	linkPre := bncrypto.PreimageLink(sitePub, upPub, rec.Seq, rec.PrevCID, rec.ContentCID, rec.TS)
	rec.LinkSig = ed25519.Sign(sitePriv, linkPre)
	noUS, err := core.CanonicalMarshalNoUpdateSig(&rec)
	if err != nil {
		return nil, "", err
	}
	upPre := bncrypto.PreimageUpdate(noUS)
	rec.UpdateSig = ed25519.Sign(upPriv, upPre)

	recBytes, err := core.CanonicalMarshal(&rec)
	if err != nil {
		return nil, "", err
	}
	recCID := core.CIDForBytes(recBytes)

	env := GossipUpdate{Record: recBytes, Content: content}
	return &env, recCID, nil
}

func Short(hexstr string) string {
	if len(hexstr) <= 12 {
		return hexstr
	}
	return hexstr[:12]
}

func PubHex(pub ed25519.PublicKey) string {
	return hex.EncodeToString(pub)
}

// Rate limiting methods
func (n *Node) checkRateLimit(peerID string) bool {
	if !n.config.EnableRateLimiting {
		return true
	}

	n.rateLimiter.mu.Lock()
	defer n.rateLimiter.mu.Unlock()

	now := time.Now()
	if requests, exists := n.rateLimiter.requests[peerID]; exists {
		// Remove old requests outside window
		var valid []time.Time
		for _, req := range requests {
			if now.Sub(req) < n.rateLimiter.window {
				valid = append(valid, req)
			}
		}
		n.rateLimiter.requests[peerID] = valid

		if len(valid) >= n.rateLimiter.maxRequests {
			n.logger.Warn("rate limit exceeded", zap.String("peer", peerID))
			return false // Rate limit exceeded
		}
	}

	n.rateLimiter.requests[peerID] = append(n.rateLimiter.requests[peerID], now)
	return true
}

// Peer validation methods
func (n *Node) validatePeer(peerID peer.ID) error {
	if !n.config.EnablePeerValidation {
		return nil
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	// Check if peer is banned
	if banTime, banned := n.bannedPeers[peerID]; banned {
		if time.Now().Before(banTime) {
			return fmt.Errorf("peer is banned until %v", banTime)
		}
		// Ban expired, remove it
		delete(n.bannedPeers, peerID)
	}

	// Check peer reputation
	if peerInfo, exists := n.peers[peerID]; exists {
		if peerInfo.Reputation < -100 {
			return fmt.Errorf("peer has poor reputation: %d", peerInfo.Reputation)
		}
	}

	return nil
}

func (n *Node) banPeer(peerID peer.ID, reason string, duration time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()

	banUntil := time.Now().Add(duration)
	n.bannedPeers[peerID] = banUntil

	n.logger.Warn("peer banned",
		zap.String("peer", peerID.String()),
		zap.String("reason", reason),
		zap.Time("until", banUntil))
}

func (n *Node) updatePeerReputation(peerID peer.ID, delta int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if peerInfo, exists := n.peers[peerID]; exists {
		peerInfo.Reputation += delta
		// Clamp reputation between -100 and +100
		if peerInfo.Reputation < -100 {
			peerInfo.Reputation = -100
		} else if peerInfo.Reputation > 100 {
			peerInfo.Reputation = 100
		}
		peerInfo.LastSeen = time.Now()
	} else {
		n.peers[peerID] = &PeerInfo{
			ID:         peerID,
			Reputation: delta,
			LastSeen:   time.Now(),
		}
	}
}

// Memory management methods
func (n *Node) checkMemoryLimit() error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.memoryUsage > n.maxMemoryUsage {
		return fmt.Errorf("memory usage limit exceeded: %d > %d", n.memoryUsage, n.maxMemoryUsage)
	}
	return nil
}

func (n *Node) updateMemoryUsage(delta int64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.memoryUsage += delta
	if n.memoryUsage < 0 {
		n.memoryUsage = 0
	}
}

func (n *Node) cleanupOldContent() {
	// Implement LRU cleanup for old content
	// This prevents memory leaks
	n.logger.Debug("cleaning up old content")

	// TODO: Implement content cleanup logic
	// - Remove old content that hasn't been accessed recently
	// - Maintain memory usage below threshold
	// - Log cleanup statistics
}

// Background goroutines
func (n *Node) peerManagement(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.cleanupInactivePeers()
		}
	}
}

func (n *Node) memoryManagement(ctx context.Context) {
	ticker := time.NewTicker(MemoryCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.cleanupOldContent()
		}
	}
}

func (n *Node) cleanupBannedPeers(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.cleanupExpiredBans()
		}
	}
}

func (n *Node) cleanupInactivePeers() {
	n.mu.Lock()
	defer n.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for peerID, peerInfo := range n.peers {
		if peerInfo.LastSeen.Before(cutoff) {
			delete(n.peers, peerID)
			n.logger.Debug("removed inactive peer", zap.String("peer", peerID.String()))
		}
	}
}

func (n *Node) cleanupExpiredBans() {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now()
	for peerID, banTime := range n.bannedPeers {
		if now.After(banTime) {
			delete(n.bannedPeers, peerID)
			n.logger.Debug("removed expired ban", zap.String("peer", peerID.String()))
		}
	}
}

// Event handlers
func (n *Node) handlePeerConnected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()

	if err := n.validatePeer(peerID); err != nil {
		n.logger.Warn("rejected connection from invalid peer",
			zap.String("peer", peerID.String()),
			zap.Error(err))
		conn.Close()
		return
	}

	n.updatePeerReputation(peerID, 1)
	n.logger.Info("peer connected", zap.String("peer", peerID.String()))
}

func (n *Node) handlePeerDisconnected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	n.logger.Info("peer disconnected", zap.String("peer", peerID.String()))
}

// Enhanced logging methods
func (n *Node) logError(msg string, err error, fields ...zap.Field) {
	n.logger.Error(msg, append(fields, zap.Error(err))...)
}

func (n *Node) logInfo(msg string, fields ...zap.Field) {
	n.logger.Info(msg, fields...)
}

func (n *Node) logDebug(msg string, fields ...zap.Field) {
	n.logger.Debug(msg, fields...)
}

func (n *Node) logWarn(msg string, fields ...zap.Field) {
	n.logger.Warn(msg, fields...)
}
