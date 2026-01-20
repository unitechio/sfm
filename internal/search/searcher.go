package search

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/owner/secure-file-manager/internal/storage"
	"github.com/owner/secure-file-manager/pkg/models"
)

type SearchResult struct {
	Path        string
	FileName    string
	FileSize    int64
	IsDirectory bool
	MatchScore  float64
}

type Searcher struct{}

func NewSearcher() *Searcher {
	return &Searcher{}
}

// SearchByName searches files by name pattern
func (s *Searcher) SearchByName(pattern string, caseSensitive bool) ([]SearchResult, error) {
	db := storage.DB()

	var indices []models.SearchIndex
	query := db.Model(&models.SearchIndex{})

	if caseSensitive {
		query = query.Where("file_name LIKE ?", "%"+pattern+"%")
	} else {
		query = query.Where("LOWER(file_name) LIKE LOWER(?)", "%"+pattern+"%")
	}

	if err := query.Find(&indices).Error; err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]SearchResult, 0, len(indices))
	for _, idx := range indices {
		score := calculateMatchScore(idx.FileName, pattern)
		results = append(results, SearchResult{
			Path:        idx.Path,
			FileName:    idx.FileName,
			FileSize:    idx.FileSize,
			IsDirectory: idx.IsDirectory,
			MatchScore:  score,
		})
	}

	return results, nil
}

// SearchByRegex searches files using regex pattern
func (s *Searcher) SearchByRegex(pattern string) ([]SearchResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	db := storage.DB()
	var indices []models.SearchIndex
	if err := db.Find(&indices).Error; err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0)
	for _, idx := range indices {
		if re.MatchString(idx.FileName) || re.MatchString(idx.Path) {
			results = append(results, SearchResult{
				Path:        idx.Path,
				FileName:    idx.FileName,
				FileSize:    idx.FileSize,
				IsDirectory: idx.IsDirectory,
				MatchScore:  1.0,
			})
		}
	}

	return results, nil
}

// SearchByExtension searches files by extension
func (s *Searcher) SearchByExtension(ext string) ([]SearchResult, error) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	db := storage.DB()
	var indices []models.SearchIndex

	if err := db.Where("file_name LIKE ?", "%"+ext).Find(&indices).Error; err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(indices))
	for _, idx := range indices {
		if filepath.Ext(idx.FileName) == ext {
			results = append(results, SearchResult{
				Path:        idx.Path,
				FileName:    idx.FileName,
				FileSize:    idx.FileSize,
				IsDirectory: idx.IsDirectory,
				MatchScore:  1.0,
			})
		}
	}

	return results, nil
}

// SearchBySize searches files by size range
func (s *Searcher) SearchBySize(minSize, maxSize int64) ([]SearchResult, error) {
	db := storage.DB()
	var indices []models.SearchIndex

	query := db.Model(&models.SearchIndex{}).Where("is_directory = ?", false)
	if minSize > 0 {
		query = query.Where("file_size >= ?", minSize)
	}
	if maxSize > 0 {
		query = query.Where("file_size <= ?", maxSize)
	}

	if err := query.Find(&indices).Error; err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(indices))
	for _, idx := range indices {
		results = append(results, SearchResult{
			Path:        idx.Path,
			FileName:    idx.FileName,
			FileSize:    idx.FileSize,
			IsDirectory: idx.IsDirectory,
			MatchScore:  1.0,
		})
	}

	return results, nil
}

func calculateMatchScore(filename, pattern string) float64 {
	filename = strings.ToLower(filename)
	pattern = strings.ToLower(pattern)

	// Exact match
	if filename == pattern {
		return 1.0
	}

	// Starts with pattern
	if strings.HasPrefix(filename, pattern) {
		return 0.9
	}

	// Contains pattern
	if strings.Contains(filename, pattern) {
		return 0.7
	}

	// Fuzzy match (simple implementation)
	score := 0.0
	for _, char := range pattern {
		if strings.ContainsRune(filename, char) {
			score += 0.1
		}
	}

	return score
}
