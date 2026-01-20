package sync

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

type P2PNode struct {
	host      host.Host
	dht       *dht.IpfsDHT
	ctx       context.Context
	cancel    context.CancelFunc
	dataDir   string
	accountID string
}

// NewP2PNode creates a new P2P node
func NewP2PNode(ctx context.Context, listenPort int, dataDir, accountID string) (*P2PNode, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Load or generate private key
	privKey, err := loadOrGenerateKey(filepath.Join(dataDir, "peer.key"))
	if err != nil {
		return nil, err
	}

	// Create listen address
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)
	if listenPort == 0 {
		listenAddr = "/ip4/0.0.0.0/tcp/0"
	}

	// Create libp2p host
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.DefaultTransports,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	nodeCtx, cancel := context.WithCancel(ctx)

	node := &P2PNode{
		host:      h,
		ctx:       nodeCtx,
		cancel:    cancel,
		dataDir:   dataDir,
		accountID: accountID,
	}

	return node, nil
}

// Start starts the P2P node
func (n *P2PNode) Start(bootstrapPeers []string, enableMDNS bool) error {
	// Setup DHT
	dhtInstance, err := dht.New(n.ctx, n.host, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}
	n.dht = dhtInstance

	// Bootstrap DHT
	if err := n.dht.Bootstrap(n.ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers
	for _, peerAddr := range bootstrapPeers {
		addr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			continue
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			continue
		}

		if err := n.host.Connect(n.ctx, *peerInfo); err != nil {
			// Log but don't fail
			continue
		}
	}

	// Setup mDNS discovery if enabled
	if enableMDNS {
		if err := n.setupMDNS(); err != nil {
			return fmt.Errorf("failed to setup mDNS: %w", err)
		}
	}

	return nil
}

// Stop stops the P2P node
func (n *P2PNode) Stop() error {
	n.cancel()
	if n.dht != nil {
		if err := n.dht.Close(); err != nil {
			return err
		}
	}
	return n.host.Close()
}

// GetPeerID returns the node's peer ID
func (n *P2PNode) GetPeerID() peer.ID {
	return n.host.ID()
}

// GetAddresses returns the node's listen addresses
func (n *P2PNode) GetAddresses() []multiaddr.Multiaddr {
	return n.host.Addrs()
}

// GetHost returns the libp2p host
func (n *P2PNode) GetHost() host.Host {
	return n.host
}

// GetDHT returns the DHT instance
func (n *P2PNode) GetDHT() *dht.IpfsDHT {
	return n.dht
}

func (n *P2PNode) setupMDNS() error {
	notifee := &discoveryNotifee{node: n}
	service := mdns.NewMdnsService(n.host, "_sfm._tcp", notifee)
	return service.Start()
}

type discoveryNotifee struct {
	node *P2PNode
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	// Auto-connect to discovered peers
	if err := n.node.host.Connect(n.node.ctx, pi); err != nil {
		// Ignore connection errors
	}
}

func loadOrGenerateKey(keyPath string) (crypto.PrivKey, error) {
	// Try to load existing key
	if data, err := os.ReadFile(keyPath); err == nil {
		return crypto.UnmarshalPrivateKey(data)
	}

	// Generate new key
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Save key
	keyData, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key: %w", err)
	}

	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	return privKey, nil
}
