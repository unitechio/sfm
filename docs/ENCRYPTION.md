# Encryption Specification

## Overview

SecureFileManager uses **AES-256-GCM** for encryption with **Argon2id** for key derivation, providing state-of-the-art security for file and folder protection.

## Key Derivation

### Argon2id Parameters

```
Algorithm: Argon2id
Time Cost: 3 iterations
Memory Cost: 64 MB (65536 KB)
Parallelism: 4 threads
Salt Size: 32 bytes (256 bits)
Output Key Size: 32 bytes (256 bits)
```

**Why Argon2id?**
- Winner of Password Hashing Competition (2015)
- Memory-hard algorithm (resistant to GPU/ASIC attacks)
- Hybrid mode combining Argon2i and Argon2d
- Recommended by OWASP for password hashing

### Salt Generation

- Cryptographically secure random salt (32 bytes)
- Unique salt per container
- Stored in container header (not secret)

## Encryption Algorithm

### AES-256-GCM

```
Algorithm: AES-256-GCM
Key Size: 256 bits
Nonce Size: 12 bytes (96 bits)
Tag Size: 16 bytes (128 bits)
```

**Why AES-256-GCM?**
- NIST-approved authenticated encryption
- Provides both confidentiality and integrity
- Hardware acceleration on modern CPUs (AES-NI)
- Resistant to padding oracle attacks

### Nonce Generation

- Random nonce per encryption operation
- Never reused with the same key
- Prepended to ciphertext

## Container Format

```
[Header: 64 bytes]
  - Magic: "SFM\x00" (4 bytes)
  - Version: 1 (4 bytes)
  - Salt: 32 bytes
  - Argon2 Time: 4 bytes
  - Argon2 Memory: 4 bytes
  - Argon2 Threads: 1 byte
  - Reserved: 15 bytes

[Encrypted Data]
  - Nonce: 12 bytes
  - Ciphertext: variable
  - Auth Tag: 16 bytes (embedded in GCM)
```

## Security Analysis

### Threat Model

**Protected Against:**
- Brute force attacks (Argon2id memory-hardness)
- Dictionary attacks (strong key derivation)
- Tampering (GCM authentication)
- Chosen-ciphertext attacks (AEAD)

**Not Protected Against:**
- Weak passwords (user responsibility)
- Keyloggers/malware on host system
- Physical access to unlocked system
- Side-channel attacks (timing, power analysis)

### Key Security Properties

1. **Confidentiality**: AES-256 provides 2^256 keyspace
2. **Integrity**: GCM authentication tag prevents tampering
3. **Forward Secrecy**: Each container uses unique salt
4. **Non-malleability**: Authenticated encryption prevents modification

## Best Practices

### Password Requirements

Recommended minimum:
- Length: 12+ characters
- Mix of uppercase, lowercase, numbers, symbols
- Avoid common words/patterns
- Use password manager

### Operational Security

1. **Never reuse passwords** across containers
2. **Securely delete** original files after encryption
3. **Backup** encrypted containers (they're safe to store anywhere)
4. **Test decryption** before deleting originals
5. **Keep software updated** for security patches

## Performance

### Benchmarks (approximate)

```
Argon2id Key Derivation: ~500ms (64MB memory)
AES-256-GCM Encryption: ~500 MB/s (with AES-NI)
File Compression: ~100 MB/s (gzip level 6)
```

### Optimization

- Hardware AES acceleration (AES-NI) when available
- Parallel compression for large files
- Streaming encryption for memory efficiency

## Compliance

- **FIPS 140-2**: AES-256 approved
- **NIST SP 800-38D**: GCM mode specification
- **RFC 9106**: Argon2 specification
- **OWASP**: Recommended cryptographic practices

## References

- [NIST AES](https://csrc.nist.gov/publications/detail/fips/197/final)
- [RFC 9106 - Argon2](https://www.rfc-editor.org/rfc/rfc9106.html)
- [NIST SP 800-38D - GCM](https://csrc.nist.gov/publications/detail/sp/800-38d/final)
