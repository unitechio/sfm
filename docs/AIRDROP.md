# LAN AirDrop Feature

## Overview

Simplified AirDrop-style file transfer for LAN (Local Area Network) without requiring accounts, NAT traversal, or internet connectivity.

## Architecture

```
┌──────────┐      ┌──────────┐
│ Device A │◄────►│ Device B │
│ (Sender) │ P2P  │ (Recv)   │
└──────────┘      └──────────┘
```

**No central server required!**

## How It Works

### 1. Discovery (mDNS)

Devices broadcast their presence on the local network using mDNS:

```json
{
  "device_name": "Dat-Laptop",
  "port": 53317,
  "capability": ["file-transfer"]
}
```

### 2. Connection (HTTP)

Direct HTTP connection between devices on the same network.

### 3. Transfer Flow

```
1. Device A scans LAN → finds Device B
2. Device A selects file
3. Device A sends transfer request (metadata)
4. Device B accepts/rejects
5. Device A streams file via HTTP
6. Device B saves and verifies
```

## Usage

### Start AirDrop Server

```bash
# Start server (auto-accept mode)
sfm airdrop start --auto-accept

# Start server (manual accept)
sfm airdrop start --name "My Laptop"
```

### List Devices on LAN

```bash
sfm airdrop list
```

Output:
```
NAME            IP              PORT
Work-PC         192.168.1.100   53317
Phone           192.168.1.101   53317
```

### Send File

```bash
sfm airdrop send 192.168.1.100 document.pdf
```

## Features

✅ **No Account Required** - Works immediately on LAN  
✅ **Auto-Discovery** - Finds devices automatically via mDNS  
✅ **Accept/Reject** - Receiver can accept or reject transfers  
✅ **Progress Tracking** - Real-time progress display  
✅ **Simple HTTP** - Easy to debug and understand  

## Comparison

| Feature | LAN AirDrop | P2P Sync |
|---------|-------------|----------|
| Discovery | mDNS | DHT |
| Transport | HTTP | libp2p |
| Account | ❌ No | ✅ Yes |
| NAT Traversal | ❌ No | ✅ Yes |
| Internet | ❌ No | ✅ Yes |
| Complexity | Low | High |

## Technical Details

### mDNS Service

- **Service Name**: `_sfm-airdrop._tcp`
- **Domain**: `local.`
- **Port**: 53317 (default, configurable)

### HTTP Endpoints

- `GET /ping` - Health check
- `POST /request` - Request file transfer
- `POST /send` - Stream file data

### File Metadata

```json
{
  "name": "photo.jpg",
  "size": 2048394,
  "mime": "image/jpeg"
}
```

## Example Session

**Device A (Sender):**
```bash
$ sfm airdrop list
Scanning for devices on LAN...
NAME            IP              PORT
Work-PC         192.168.1.100   53317

$ sfm airdrop send 192.168.1.100 photo.jpg
Sending photo.jpg to 192.168.1.100:53317...
Progress: 100.00%
✓ File sent successfully
```

**Device B (Receiver):**
```bash
$ sfm airdrop start

AirDrop server started
Device: Work-PC
Port: 53317
Downloads: C:\Users\user\.sfm\airdrop

Incoming file transfer:
  File: photo.jpg
  Size: 2048394 bytes
  From: 192.168.1.101:52341
Accept? (y/n): y

Receiving photo.jpg: 100.00%
✓ Saved to: C:\Users\user\.sfm\airdrop\photo.jpg
```

## Advantages Over P2P Sync

1. **Simpler** - No pairing, no accounts
2. **Faster Setup** - Works immediately
3. **Easier Debug** - Standard HTTP
4. **Lower Latency** - Direct connection on LAN
5. **No Dependencies** - No STUN/TURN servers

## Use Cases

- Quick file sharing between laptops on same WiFi
- Office file transfers without cloud
- Development team file sharing
- Home network file transfers

## Future Enhancements

- [ ] Encryption (TLS)
- [ ] Multiple file selection
- [ ] Folder transfer
- [ ] Resume support
- [ ] QR code for easy connection
- [ ] GUI interface

## Security Note

⚠️ **LAN AirDrop is designed for trusted networks only**

- No encryption by default (HTTP)
- No authentication
- Anyone on the network can discover your device
- Only use on trusted networks (home, office)

For internet transfers or untrusted networks, use the P2P Sync feature instead.
