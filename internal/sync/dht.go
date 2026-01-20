package sync

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/owner/secure-file-manager/internal/storage"
	"github.com/owner/secure-file-manager/pkg/models"
)

type DHTManager struct {
	node *P2PNode
}

func NewDHTManager(node *P2PNode) *DHTManager {
	return &DHTManager{node: node}
}

// AdvertiseAccount advertises this device on the DHT
func (dm *DHTManager) AdvertiseAccount(ctx context.Context, accountID string) error {
	// This is a simplified version - in production would use proper DHT advertising
	// For now, we'll just store in local database
	db := storage.DB()

	var accountInfo models.AccountInfo
	result := db.Where("account_id = ?", accountID).FirstOrCreate(&accountInfo, models.AccountInfo{
		AccountID:  accountID,
		DeviceName: "Local Device",
		PeerID:     dm.node.GetPeerID().String(),
	})

	return result.Error
}

// DiscoverPeers discovers peers with the same account ID
func (dm *DHTManager) DiscoverPeers(ctx context.Context, accountID string) ([]peer.AddrInfo, error) {
	// In production, would query DHT for peers advertising the same account ID
	// For now, return paired devices from database
	db := storage.DB()

	var devices []models.PairedDevice
	if err := db.Where("account_id = ?", accountID).Find(&devices).Error; err != nil {
		return nil, err
	}

	peers := make([]peer.AddrInfo, 0, len(devices))
	for _, device := range devices {
		peerID, err := peer.Decode(device.PeerID)
		if err != nil {
			continue
		}

		peers = append(peers, peer.AddrInfo{
			ID: peerID,
		})
	}

	return peers, nil
}

// UpdatePeerStatus updates the online status of paired devices
func (dm *DHTManager) UpdatePeerStatus(ctx context.Context) error {
	db := storage.DB()

	var devices []models.PairedDevice
	if err := db.Find(&devices).Error; err != nil {
		return err
	}

	for _, device := range devices {
		peerID, err := peer.Decode(device.PeerID)
		if err != nil {
			continue
		}

		// Check if peer is connected
		conns := dm.node.host.Network().ConnsToPeer(peerID)
		isOnline := len(conns) > 0

		db.Model(&device).Updates(map[string]interface{}{
			"is_online": isOnline,
			"last_seen": time.Now(),
		})
	}

	return nil
}

// StartPeriodicAdvertisement starts periodic DHT advertisement
func (dm *DHTManager) StartPeriodicAdvertisement(ctx context.Context, accountID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dm.AdvertiseAccount(ctx, accountID)
			dm.UpdatePeerStatus(ctx)
		}
	}
}
