# SecureFileManager

Cross-platform file management tool with encryption, fast search, P2P sync, and LAN file sharing.

## Features

### 🔐 File Encryption
- AES-256-GCM + Argon2id key derivation
- Password-protected containers
- Cross-platform (Windows & Linux)

### 🔍 Fast File Search
- Concurrent indexing
- Multiple search modes (name, regex, extension, size)
- Relevance scoring

### 🔄 P2P File Sync
- Fully decentralized (libp2p + DHT)
- Device pairing via QR code
- End-to-end encryption
- NAT traversal

### ⚡ LAN AirDrop
- Zero-config LAN file sharing
- Auto-discovery (mDNS)
- Accept/reject transfers
- No account required

## Quick Start

```bash
# Encrypt
sfm encrypt file.txt -o encrypted.sfm

# Search
sfm search index /path
sfm search query "document"

# P2P Sync
sfm sync pair --qr code.png
sfm sync send "Device" file.pdf

# LAN AirDrop
sfm airdrop start
sfm airdrop list
sfm airdrop send 192.168.1.100 file.pdf
```

## Installation

```bash
go build -o sfm.exe ./cmd/sfm
```

## Documentation

- [README](README.md) - This file
- [ENCRYPTION.md](docs/ENCRYPTION.md) - Encryption spec
- [P2P_PROTOCOL.md](docs/P2P_PROTOCOL.md) - P2P protocol
- [AIRDROP.md](docs/AIRDROP.md) - LAN AirDrop guide
- [BUILD.md](docs/BUILD.md) - Build instructions

## License

MIT
