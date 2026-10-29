// Package cache reads and writes the on-disk quote cache.
package cache

import (
	"encoding/json"
	"os"
	"time"
)

const (
	maxAge     = 12 * time.Hour
	timeLayout = "2006-01-02 15-04-05"
)

type Symbol struct {
	Name  string  `json:"Name"`
	Price float64 `json:"Price"`
}

type file struct {
	Date    string   `json:"Date"`
	Symbols []Symbol `json:"Symbols"`
}

// Returns cached symbols when the file is less than maxAge.
func ReadFreshCache(path string, now time.Time) ([]Symbol, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	var cached file
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false, err
	}
	updatedAt, err := time.Parse(timeLayout, cached.Date)
	if err != nil {
		return nil, false, err
	}
	age := now.Sub(updatedAt)
	if age < 0 || age >= maxAge {
		return nil, false, nil
	}
	return cached.Symbols, true, nil
}

// WriteCache writes symbols and their current timestamp to path.
func WriteCache(path string, symbols []Symbol) error {
	cached := file{Date: time.Now().Format(timeLayout), Symbols: symbols}
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
