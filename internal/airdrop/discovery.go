package airdrop

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/hashicorp/mdns"
)

const (
	ServiceName = "_sfm-airdrop._tcp"
	Domain      = "local."
)

type DeviceInfo struct {
	Name      string
	IP        net.IP
	Port      int
	Hostname  string
	Timestamp time.Time
}

type Discovery struct {
	deviceName string
	port       int
	devices    map[string]*DeviceInfo
	server     *mdns.Server
}

func NewDiscovery(deviceName string, port int) *Discovery {
	return &Discovery{
		deviceName: deviceName,
		port:       port,
		devices:    make(map[string]*DeviceInfo),
	}
}

// StartAdvertising broadcasts this device on the network
func (d *Discovery) StartAdvertising() error {
	host, err := getHostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	info := []string{
		fmt.Sprintf("name=%s", d.deviceName),
		fmt.Sprintf("capability=file-transfer"),
	}

	service, err := mdns.NewMDNSService(
		host,
		ServiceName,
		Domain,
		"",
		d.port,
		nil,
		info,
	)
	if err != nil {
		return fmt.Errorf("failed to create mDNS service: %w", err)
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return fmt.Errorf("failed to create mDNS server: %w", err)
	}

	d.server = server
	log.Printf("Broadcasting as '%s' on port %d", d.deviceName, d.port)
	return nil
}

// StopAdvertising stops broadcasting
func (d *Discovery) StopAdvertising() error {
	if d.server != nil {
		return d.server.Shutdown()
	}
	return nil
}

// ScanDevices scans for other devices on the network
func (d *Discovery) ScanDevices(ctx context.Context, duration time.Duration) ([]*DeviceInfo, error) {
	entriesCh := make(chan *mdns.ServiceEntry, 10)

	// Clear previous devices
	d.devices = make(map[string]*DeviceInfo)

	// Start scanning
	go func() {
		params := &mdns.QueryParam{
			Service:             ServiceName,
			Domain:              Domain,
			Timeout:             duration,
			Entries:             entriesCh,
			WantUnicastResponse: false,
		}
		mdns.Query(params)
		close(entriesCh)
	}()

	// Collect entries
	for entry := range entriesCh {
		if entry.AddrV4 == nil {
			continue
		}

		// Parse device info from TXT records
		deviceName := d.deviceName // default
		for _, txt := range entry.InfoFields {
			if len(txt) > 5 && txt[:5] == "name=" {
				deviceName = txt[5:]
			}
		}

		// Skip self
		if deviceName == d.deviceName {
			continue
		}

		device := &DeviceInfo{
			Name:      deviceName,
			IP:        entry.AddrV4,
			Port:      entry.Port,
			Hostname:  entry.Host,
			Timestamp: time.Now(),
		}

		d.devices[device.IP.String()] = device
	}

	// Convert to slice
	result := make([]*DeviceInfo, 0, len(d.devices))
	for _, device := range d.devices {
		result = append(result, device)
	}

	return result, nil
}

// GetDevices returns currently known devices
func (d *Discovery) GetDevices() []*DeviceInfo {
	result := make([]*DeviceInfo, 0, len(d.devices))
	for _, device := range d.devices {
		result = append(result, device)
	}
	return result
}

func getHostname() (string, error) {
	hostname, err := net.LookupHost("localhost")
	if err != nil {
		return "unknown", nil
	}
	if len(hostname) > 0 {
		return hostname[0], nil
	}
	return "unknown", nil
}
