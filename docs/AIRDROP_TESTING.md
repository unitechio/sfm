# Secure LAN AirDrop - Testing Guide

## What's New

✅ **Priority 1 - Security:**
- Device identity (Ed25519 keypair)
- SHA256 fingerprint
- Handshake protocol with signatures
- Device info shown before accept
- E2E encryption (ECDH + AES-GCM)
- Encrypted chunks

✅ **Priority 2 - Reliability:**
- Chunk protocol (4MB chunks)
- SHA256 checksum per chunk
- ACK mechanism
- Resume capability
- Flow control

## Testing

### 1. Start Secure Server

```bash
./sfm.exe airdrop start
```

**Expected output:**
```
Device fingerprint: a1:b2:c3:d4:...
🔐 Secure AirDrop server started
Device: VDTC-DATPT
Port: 53317
Downloads: C:\Users\ad-vdtc\.sfm\airdrop
```

### 2. Send File (Same Machine Test)

```bash
# In another terminal
./sfm.exe airdrop send 127.0.0.1 README.md
```

**Expected output:**
```
Client fingerprint: a1:b2:c3:d4:...
🔐 Sending README.md to 127.0.0.1:53317...
Handshake accepted. Session ID: uuid-here
Sending 1 chunks...
Progress: 100.00% (1/1 chunks)
✓ File sent successfully
```

**Server side:**
```
📥 Incoming file transfer:
  From: VDTC-DATPT
  Fingerprint: a1:b2:c3:d4:...
  File: README.md
  Size: 1234 bytes

Accept? (y/n): y
Receiving README.md: 100.00% (1/1 chunks)
✓ Saved to: C:\Users\ad-vdtc\.sfm\airdrop\README.md
```

### 3. Verify Security

**Check device identity:**
```bash
ls ~/.sfm/airdrop/
# Should see: device.pub, device.key
```

**Fingerprint is consistent:**
- Same fingerprint on every run
- Derived from device.pub

**Encryption verified:**
- Chunks are encrypted in transit
- Only decrypted on receiver
- Session key unique per transfer

### 4. Test Large File (Chunking)

```bash
# Create 20MB test file
dd if=/dev/zero of=test20mb.bin bs=1M count=20

# Send it
./sfm.exe airdrop send 127.0.0.1 test20mb.bin
```

**Expected:**
- File split into 5 chunks (4MB each)
- Progress shows chunk count
- Each chunk verified with checksum
- ACK received for each chunk

### 5. Test Resume (Future)

Currently implemented but needs testing:
1. Start transfer
2. Kill sender mid-transfer
3. Restart transfer
4. Should resume from last ACK'd chunk

## Security Features

### 1. Device Identity
- Ed25519 keypair generated once
- Stored in `~/.sfm/airdrop/device.{pub,key}`
- Fingerprint: SHA256 of public key

### 2. Handshake Protocol
```
1. Sender → Receiver: HandshakeRequest
   - Device name
   - Fingerprint
   - Ephemeral public key (X25519)
   - File metadata
   - Signature (Ed25519)

2. Receiver → Sender: HandshakeResponse
   - Accepted/Rejected
   - Ephemeral public key
   - Session ID
```

### 3. E2E Encryption
- ECDH key exchange (X25519)
- Derive AES-256 session key
- Each chunk encrypted with AES-GCM
- Unique nonce per chunk

### 4. Integrity
- SHA256 checksum per chunk
- Verified before writing
- Failed chunks rejected

## Comparison: Before vs After

| Feature | Before | After |
|---------|--------|-------|
| Identity | ❌ None | ✅ Ed25519 |
| Fingerprint | ❌ None | ✅ SHA256 |
| Handshake | ❌ Direct send | ✅ Signed request |
| Encryption | ❌ Plain HTTP | ✅ E2E (ECDH+AES) |
| Chunking | ❌ Single POST | ✅ 4MB chunks |
| Checksum | ❌ None | ✅ SHA256/chunk |
| Resume | ❌ No | ✅ Yes |
| ACK | ❌ No | ✅ Per chunk |

## Next Steps (Priority 3 - UX)

- [ ] Better progress UI
- [ ] Pause/cancel support
- [ ] Desktop notifications
- [ ] Choose save location
- [ ] GUI (optional)

## Migration to Internet P2P

**Already reusable:**
- ✅ Chunk protocol
- ✅ E2E encryption
- ✅ Handshake flow
- ✅ ACK mechanism

**Just replace:**
- mDNS → Signaling server
- HTTP → WebRTC DataChannel
