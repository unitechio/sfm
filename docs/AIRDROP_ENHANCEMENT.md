# LAN AirDrop Enhancement Plan

## Current Status

✅ **Working:**
- mDNS discovery
- HTTP server/client
- Basic file transfer
- Progress tracking

❌ **Missing Critical Features:**
- Device identity & trust
- Handshake protocol
- E2E encryption
- Chunk protocol with resume
- Flow control

## Priority 1: Security & Trust

### 1.1 Device Identity

**Implementation:**
- Generate Ed25519 keypair per device
- Create SHA256 fingerprint
- Store in `~/.sfm/airdrop/device.{pub,key}`

**Files:**
- `internal/airdrop/identity.go` ✅ Created

**Benefits:**
- Unique device identification
- Cryptographic verification
- User can verify fingerprint

### 1.2 Handshake Protocol

**Flow:**
```
Sender                    Receiver
  |                          |
  |--- HandshakeRequest ---->|
  |    (fingerprint, file)   |
  |                          |
  |                    [Show UI]
  |                    [User Accept/Reject]
  |                          |
  |<-- HandshakeResponse ----|
  |    (accepted + pubkey)   |
  |                          |
  |--- Encrypted Chunks ---->|
```

**Data Structure:**
```json
{
  "device_name": "Laptop",
  "device_fingerprint": "a1:b2:c3:...",
  "ephemeral_pubkey": "...",
  "file_metadata": {...},
  "signature": "..."
}
```

**Files:**
- `internal/airdrop/protocol.go` ✅ Created

### 1.3 E2E Encryption

**Key Exchange:**
1. ECDH with ephemeral X25519 keys
2. Derive AES-256 session key
3. Encrypt each chunk with AES-GCM

**Files:**
- `internal/airdrop/encryption.go` ✅ Created

**Security:**
- Forward secrecy (ephemeral keys)
- Authenticated encryption (GCM)
- Per-transfer session keys

## Priority 2: Reliability

### 2.1 Chunk Protocol

**Chunk Structure:**
```json
{
  "index": 12,
  "total": 340,
  "size": 4194304,
  "checksum": "sha256...",
  "session_id": "uuid"
}
```

**Benefits:**
- Resume from any chunk
- Verify integrity per chunk
- Parallel chunk transfer (future)

### 2.2 Flow Control

**Sliding Window:**
```
Window Size: 10 chunks
Send chunks 0-9 → Wait for ACKs → Send 10-19
```

**ACK Protocol:**
```json
{
  "index": 5,
  "session_id": "uuid",
  "success": true
}
```

**Benefits:**
- Prevent memory overflow
- Adaptive to network speed
- Better error handling

### 2.3 Resume Capability

**Transfer State:**
```json
{
  "session_id": "uuid",
  "total_chunks": 340,
  "received_chunks": [0,1,2,5,6,7],
  "progress": 35.2,
  "can_resume": true
}
```

**Resume Flow:**
1. Client requests transfer status
2. Server returns received chunks
3. Client sends only missing chunks

## Implementation Order

### Phase 1: Security (This Phase)
- [x] Device identity generation
- [x] Handshake protocol design
- [x] E2E encryption primitives
- [ ] Update server to use handshake
- [ ] Update client to use handshake
- [ ] Add fingerprint verification UI

### Phase 2: Chunking
- [ ] Implement chunk protocol
- [ ] Add chunk checksums
- [ ] Update transfer to use chunks
- [ ] Add chunk ACK mechanism

### Phase 3: Resume & Flow Control
- [ ] Implement transfer state tracking
- [ ] Add resume capability
- [ ] Implement sliding window
- [ ] Add retry logic

### Phase 4: UX Polish
- [ ] Better progress display
- [ ] Pause/cancel support
- [ ] Notification system
- [ ] Choose save location

## Migration Path to Internet P2P

**Reusable Components:**
- ✅ Chunk protocol → WebRTC DataChannel
- ✅ E2E encryption → Same keys
- ✅ Handshake → Signaling server
- ✅ Flow control → Same logic

**Only Replace:**
- mDNS → Signaling server
- HTTP → WebRTC transport

## Next Steps

1. Update `transfer.go` to use handshake
2. Update `client.go` to use handshake
3. Test handshake flow
4. Implement chunking
5. Add resume support
