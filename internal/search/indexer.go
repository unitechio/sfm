package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/owner/secure-file-manager/internal/storage"
	"github.com/owner/secure-file-manager/pkg/models"
)

type Indexer struct {
	maxWorkers int
	mu         sync.Mutex
}

func NewIndexer(maxWorkers int) *Indexer {
	return &Indexer{
		maxWorkers: maxWorkers,
	}
}

// IndexDirectory indexes all files in a directory
func (idx *Indexer) IndexDirectory(rootPath string) error {
	type fileInfo struct {
		path    string
		info    os.FileInfo
		relPath string
	}

	fileChan := make(chan fileInfo, 100)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < idx.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fi := range fileChan {
				if err := idx.indexFile(fi.path, fi.info, fi.relPath); err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}
			}
		}()
	}

	// Walk directory
	go func() {
		filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				errChan <- err
				return err
			}

			relPath, _ := filepath.Rel(rootPath, path)
			fileChan <- fileInfo{path, info, relPath}
			return nil
		})
		close(fileChan)
	}()

	// Wait for workers
	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (idx *Indexer) indexFile(path string, info os.FileInfo, relPath string) error {
	db := storage.DB()

	searchIndex := models.SearchIndex{
		Path:         path,
		FileName:     info.Name(),
		FileSize:     info.Size(),
		ModifiedTime: info.ModTime(),
		IsDirectory:  info.IsDir(),
	}

	// Upsert
	result := db.Where("path = ?", path).FirstOrCreate(&searchIndex)
	if result.Error != nil {
		return fmt.Errorf("failed to index file %s: %w", path, result.Error)
	}

	// Update if modified
	if result.RowsAffected == 0 {
		db.Model(&searchIndex).Updates(map[string]interface{}{
			"file_size":     info.Size(),
			"modified_time": info.ModTime(),
		})
	}

	return nil
}

// RemoveFromIndex removes a file from the index
func (idx *Indexer) RemoveFromIndex(path string) error {
	db := storage.DB()
	return db.Where("path = ?", path).Delete(&models.SearchIndex{}).Error
}

// UpdateIndex incrementally updates the index
func (idx *Indexer) UpdateIndex(rootPath string) error {
	db := storage.DB()

	// Get all indexed files
	var indexed []models.SearchIndex
	if err := db.Find(&indexed).Error; err != nil {
		return err
	}

	existingFiles := make(map[string]bool)
	for _, item := range indexed {
		existingFiles[item.Path] = true
	}

	// Check for deleted files
	for path := range existingFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			idx.RemoveFromIndex(path)
		}
	}

	// Index new/modified files
	return idx.IndexDirectory(rootPath)
}

// GetStats returns indexing statistics
func (idx *Indexer) GetStats() (int64, error) {
	db := storage.DB()
	var count int64
	err := db.Model(&models.SearchIndex{}).Count(&count).Error
	return count, err
}
