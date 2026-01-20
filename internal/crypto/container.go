package crypto

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	MagicBytes = "SFM\x00"
	Version    = 1
	HeaderSize = 64
)

// ContainerHeader represents the encrypted container header
type ContainerHeader struct {
	Magic         [4]byte
	Version       uint32
	Salt          [32]byte
	Argon2Time    uint32
	Argon2Memory  uint32
	Argon2Threads uint8
	Reserved      [15]byte
}

// CreateContainer creates an encrypted container from a file or directory
func CreateContainer(sourcePath, containerPath, password string, argon2Time, argon2Memory uint32, argon2Threads uint8) error {
	// Generate salt
	salt, err := GenerateSalt()
	if err != nil {
		return err
	}

	// Derive key
	key := DeriveKey(password, salt, argon2Time, argon2Memory, argon2Threads)

	// Create container file
	containerFile, err := os.Create(containerPath)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	defer containerFile.Close()

	// Write header
	header := ContainerHeader{
		Version:       Version,
		Argon2Time:    argon2Time,
		Argon2Memory:  argon2Memory,
		Argon2Threads: argon2Threads,
	}
	copy(header.Magic[:], MagicBytes)
	copy(header.Salt[:], salt)

	if err := binary.Write(containerFile, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Create tar.gz archive in memory
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add files to archive
	if err := addToArchive(tarWriter, sourcePath, ""); err != nil {
		return err
	}

	tarWriter.Close()
	gzWriter.Close()

	// Encrypt and write data
	if err := EncryptStream(&buf, containerFile, key); err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	return nil
}

// ExtractContainer extracts an encrypted container
func ExtractContainer(containerPath, outputPath, password string) error {
	// Open container file
	containerFile, err := os.Open(containerPath)
	if err != nil {
		return fmt.Errorf("failed to open container: %w", err)
	}
	defer containerFile.Close()

	// Read header
	var header ContainerHeader
	if err := binary.Read(containerFile, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Verify magic bytes
	if string(header.Magic[:]) != MagicBytes {
		return fmt.Errorf("invalid container format")
	}

	// Derive key
	key := DeriveKey(password, header.Salt[:], header.Argon2Time, header.Argon2Memory, header.Argon2Threads)

	// Decrypt data
	var buf bytes.Buffer
	if err := DecryptStream(containerFile, &buf, key); err != nil {
		return fmt.Errorf("failed to decrypt data (wrong password?): %w", err)
	}

	// Extract tar.gz archive
	gzReader, err := gzip.NewReader(&buf)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		target := filepath.Join(outputPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
		}
	}

	return nil
}

func addToArchive(tarWriter *tar.Writer, source, baseDir string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		if baseDir != "" {
			relPath, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			header.Name = filepath.Join(baseDir, relPath)
		} else {
			header.Name = filepath.Base(path)
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})
}
