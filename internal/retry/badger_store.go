package retry

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/user/webhook-dispatch/internal/models"
)

// BadgerStore implements Store using BadgerDB
type BadgerStore struct {
	db *badger.DB
}

// NewBadgerStore creates a new BadgerDB-backed retry store
func NewBadgerStore(dataDir string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(dataDir)
	opts.Logger = nil // Disable badger's noisy logging

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}

	// Start garbage collection
	go runGC(db)

	return &BadgerStore{db: db}, nil
}

// Save stores a retry entry with a time-based key for efficient querying
func (s *BadgerStore) Save(entry *models.RetryEntry) error {
	// Key format: retry:{timestamp}:{webhook_id}:{destination_id}
	// This allows efficient range scans for due retries
	key := fmt.Sprintf("retry:%d:%s:%s",
		entry.NextRetryAt.Unix(),
		entry.WebhookEvent.ID,
		entry.Destination.ID,
	)

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal retry entry: %w", err)
	}

	// Set TTL to 7 days
	ttl := 7 * 24 * time.Hour

	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), data).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

// Get retrieves a retry entry by ID (webhook_id:destination_id)
func (s *BadgerStore) Get(id string) (*models.RetryEntry, error) {
	var entry *models.RetryEntry

	err := s.db.View(func(txn *badger.Txn) error {
		// Scan for key with matching ID
		prefix := []byte("retry:")
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			// Extract ID from key (format: retry:timestamp:webhook_id:destination_id)
			parts := strings.Split(key, ":")
			if len(parts) == 4 {
				entryID := parts[2] + ":" + parts[3]
				if entryID == id {
					return item.Value(func(val []byte) error {
						return json.Unmarshal(val, &entry)
					})
				}
			}
		}

		return fmt.Errorf("retry entry not found: %s", id)
	})

	return entry, err
}

// GetDue retrieves all retry entries that are due for retry
func (s *BadgerStore) GetDue(before time.Time) ([]*models.RetryEntry, error) {
	var entries []*models.RetryEntry

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("retry:")

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			// Extract timestamp from key
			parts := strings.Split(key, ":")
			if len(parts) != 4 {
				continue
			}

			var timestamp int64
			if _, err := fmt.Sscanf(parts[1], "%d", &timestamp); err != nil {
				continue
			}

			// Check if due
			nextRetry := time.Unix(timestamp, 0)
			if nextRetry.After(before) {
				// Keys are sorted by timestamp, so we can stop here
				break
			}

			// Decode entry
			err := item.Value(func(val []byte) error {
				var entry models.RetryEntry
				if err := json.Unmarshal(val, &entry); err != nil {
					return err
				}
				entries = append(entries, &entry)
				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return entries, err
}

// Delete removes a retry entry by ID
func (s *BadgerStore) Delete(id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		// Scan for key with matching ID
		prefix := []byte("retry:")
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			// Extract ID from key
			parts := strings.Split(key, ":")
			if len(parts) == 4 {
				entryID := parts[2] + ":" + parts[3]
				if entryID == id {
					return txn.Delete(item.Key())
				}
			}
		}

		return fmt.Errorf("retry entry not found: %s", id)
	})
}

// Count returns the total number of retry entries
func (s *BadgerStore) Count() (int, error) {
	count := 0

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("retry:")

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}

		return nil
	})

	return count, err
}

// Close closes the BadgerDB database
func (s *BadgerStore) Close() error {
	return s.db.Close()
}

// runGC runs BadgerDB garbage collection periodically
func runGC(db *badger.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
	again:
		err := db.RunValueLogGC(0.5)
		if err == nil {
			goto again
		}
	}
}
