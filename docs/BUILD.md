# Build and Run Instructions

## Prerequisites
- Go 1.21 or higher
- Windows 10+ or Linux

## Build

```bash
cd t:\OWNER_DAT\CODE\OWNER\WINDOWN_INFO\secure-file-manager

# Install dependencies
go mod tidy

# Build executable
go build -o sfm.exe ./cmd/sfm
```

## Quick Test

### 1. Test Encryption
```bash
# Create test file
echo "Hello, SecureFileManager!" > test.txt

# Encrypt
./sfm.exe encrypt test.txt -o test.sfm
# Enter password when prompted

# Decrypt
./sfm.exe decrypt test.sfm -o decrypted
# Enter same password
```

### 2. Test Search
```bash
# Index current directory
./sfm.exe search index .

# Search for files
./sfm.exe search query "sfm"
./sfm.exe search query --ext exe
```

### 3. Test P2P Sync

**Device A:**
```bash
# Initialize and generate pairing code
./sfm.exe sync init
./sfm.exe sync pair --qr pairing.png
```

**Device B:**
```bash
# Connect using pairing code
./sfm.exe sync connect "PIN|PeerID|Address" --name "Device A"

# List devices
./sfm.exe sync devices
```

**Send file from A to B:**
```bash
./sfm.exe sync send "Device A" test.txt
```

## Configuration

Config file: `~/.sfm/config.yaml`

Edit to customize:
- Encryption parameters
- Search settings
- P2P ports and bootstrap peers

## Troubleshooting

**Build errors:**
- Run `go mod tidy` to fix dependencies
- Ensure Go version >= 1.21

**Runtime errors:**
- Check `~/.sfm/sfm.log` for details
- Ensure required ports are not blocked

## Next Steps

1. Test on Linux for cross-platform verification
2. Implement FUSE/WinFsp mounting
3. Add GUI (optional)
4. Deploy to production
