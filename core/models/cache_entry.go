package models

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"time"
)

type CacheEntry struct {
	FilePath   string      `json:"file_path"`
	ModTime    time.Time   `json:"mod_time"`
	FileHash   string      `json:"file_hash"`
	ParsedFile *ParsedFile `json:"parsed_file"`
	CreatedAt  time.Time   `json:"created_at"`
}

func NewCacheEntry(filePath string, parsedFile *ParsedFile) (*CacheEntry, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	hash, err := calculateFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash for file %s: %w", filePath, err)
	}

	return &CacheEntry{
		FilePath:   filePath,
		ModTime:    stat.ModTime(),
		FileHash:   hash,
		ParsedFile: parsedFile,
		CreatedAt:  time.Now(),
	}, nil
}

func (ce *CacheEntry) IsValid() (bool, error) {
	stat, err := os.Stat(ce.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file %s: %w", ce.FilePath, err)
	}

	if stat.ModTime().Equal(ce.ModTime) {
		return true, nil
	}

	currentHash, err := calculateFileHash(ce.FilePath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate current hash for file %s: %w", ce.FilePath, err)
	}

	if currentHash == ce.FileHash {
		ce.ModTime = stat.ModTime()
		return true, nil
	}

	return false, nil
}

func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
