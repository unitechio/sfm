# P2P File Transfer Protocol

## Overview

SecureFileManager uses a custom libp2p protocol for direct peer-to-peer file transfers with end-to-end encryption.

## Protocol Specification

### Protocol ID
```
/sfm/transfer/1.0.0
```

### Transport Layer

**libp2p Stack:**
- Transport: TCP, QUIC
- Security: TLS 1.3 (default), Noise
- Multiplexing: mplex, yamux
- NAT Traversal: AutoNAT, Circuit Relay

## Pairing Workflow

### 1. Device Pairing

**Device A (Initiator):**
```
1. Generate 8-digit PIN
2. Get Peer ID and listen addresses
3. Create pairing data: "PIN|PeerID|Address"
4. Generate QR code
5. Display to user
```

**Device B (Joiner):**
```
1. Scan QR code or enter pairing code
2. Parse peer information
3. Connect to Device A via libp2p
4. Exchange public keys
5. Derive shared Account ID
6. Store paired device in database
```

### 2. Account ID Generation

```
AccountID = Base64(Hash(PeerID_A || PeerID_B))[:32]
```

All devices paired together share the same Account ID.

## File Transfer Protocol

### Message Flow

```
Sender                          Receiver
  |                                |
  |--- Open libp2p stream -------->|
  |                                |
  |--- Send Metadata ------------->|
  |    (filename, size)            |
  |                                |
  |--- Send Encrypted Chunks ----->|
  |    (4MB per chunk)             |
  |                                |
  |--- Send Checksum ------------->|
  |    (SHA-256)                   |
  |                                |
  |<-- Close stream ---------------|
```

### Metadata Format

```
[Filename Length: 4 bytes (uint32)]
[Filename: variable UTF-8]
[File Size: 8 bytes (int64)]
```

### Chunk Format

```
[Chunk Size: 4 bytes (uint32)]
[Encrypted Data: variable]
  - Nonce: 12 bytes
  - Ciphertext: variable
  - Auth Tag: 16 bytes
```

### Checksum

```
[SHA-256 Hash: 32 bytes]
```

## Encryption

### Per-Transfer Encryption

Each file transfer uses:
- **Algorithm**: AES-256-GCM
- **Key**: Derived from shared secret (peer keys)
- **Nonce**: Random per chunk
- **Chunk Size**: 4 MB

### Key Exchange

```
1. Each device has Ed25519 keypair
2. Public keys exchanged during pairing
3. Shared secret derived via ECDH
4. Transfer key = HKDF(shared_secret, file_hash)
```

## Peer Discovery

### Local Network (mDNS)

```
Service: _sfm._tcp.local.
Port: Dynamic (libp2p assigned)
TXT Records:
  - peer_id=<PeerID>
  - account_id=<AccountID>
```

### Internet (DHT)

```
1. Bootstrap to IPFS DHT network
2. Advertise under key: /sfm/account/<AccountID>
3. Discover peers advertising same key
4. Connect via libp2p multiaddress
```

## NAT Traversal

### Strategy

1. **Direct Connection** (preferred)
   - Both peers have public IPs
   - Or both on same LAN

2. **Hole Punching**
   - Use STUN to discover public endpoint
   - Coordinate simultaneous connect
   - Works for most NAT types

3. **Relay Fallback**
   - Use libp2p circuit relay
   - Public relay nodes
   - Slower but always works

### STUN Configuration

```
STUN Servers:
  - stun.l.google.com:19302
  - stun1.l.google.com:19302
```

## Resume Support

### Chunk Tracking

```
Transfer State:
  - Total chunks: file_size / chunk_size
  - Received chunks: bitmap
  - Missing chunks: list
```

### Resume Protocol

```
1. Receiver stores partial file + state
2. On reconnect, send missing chunk list
3. Sender resumes from missing chunks
4. Verify final checksum
```

## Error Handling

### Transfer Errors

| Error | Action |
|-------|--------|
| Connection lost | Retry with exponential backoff |
| Checksum mismatch | Delete file, request retransmit |
| Disk full | Abort, notify sender |
| Decryption failed | Abort, possible key mismatch |

### Retry Policy

```
Max Retries: 3
Backoff: 1s, 2s, 4s
Timeout: 30s per chunk
```

## Performance Optimization

### Concurrent Transfers

- Multiple files: parallel streams
- Single file: sequential chunks (ordered)
- Bandwidth sharing: fair queuing

### Compression

Optional gzip compression before encryption:
- Enabled for text files
- Disabled for already compressed (images, videos)
- Adaptive based on compression ratio

## Security Considerations

### Threat Model

**Protected:**
- Eavesdropping (E2E encryption)
- MITM (TLS + key pinning)
- Tampering (GCM auth tags)
- Replay attacks (nonce uniqueness)

**Not Protected:**
- Malicious paired device
- Compromised peer keys
- Traffic analysis (metadata visible)

### Best Practices

1. **Verify pairing** - Check QR code carefully
2. **Revoke devices** - Remove untrusted pairs
3. **Secure storage** - Protect peer.key file
4. **Network security** - Use trusted networks

## Example Usage

### Send File

```bash
# Start P2P node
sfm sync init

# Pair with device
sfm sync pair --qr qr.png

# Send file
sfm sync send "Device Name" file.pdf
```

### Receive File

```bash
# Pair with sender
sfm sync connect <pairing-code>

# Files auto-received to ~/.sfm/downloads/
```

## Protocol Extensions

### Future Enhancements

- [ ] Folder sync (bidirectional)
- [ ] Delta sync (rsync-like)
- [ ] Conflict resolution
- [ ] Version history
- [ ] Selective sync (filters)
