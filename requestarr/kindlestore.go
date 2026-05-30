package requestarr

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	requestLimit  = 3
	requestPeriod = 72 * time.Hour
)

var requestLimitOverrides = map[string]int{
	"d0m182goodwin@gmail.com": 50,
}

func RequestLimit(cfEmail string) int {
	if limit, ok := requestLimitOverrides[cfEmail]; ok {
		return limit
	}
	return requestLimit
}

// BookRequest records a user's book request with a timestamp for rate limiting.
type BookRequest struct {
	CFEmail     string    `json:"cfEmail"`
	RequestedAt time.Time `json:"requestedAt"`
}

// KindleStore persists four mappings as JSON files:
//   - kindle_emails.json:   CF Access email → Kindle email
//   - book_requests.json:   LL book ID      → CF Access email (pending delivery)
//   - request_history.json: LL book ID      → BookRequest    (for rate limiting, 72h rolling)
//   - sent_books.json:      "cfEmail|bookID" → sent timestamp (to suppress re-send button)
type KindleStore struct {
	mu             sync.RWMutex
	kindleFile     string
	requestsFile   string
	historyFile    string
	sentFile       string
	kindleEmails   map[string]string
	bookRequests   map[string]string
	requestHistory map[string]BookRequest
	sentBooks      map[string]time.Time // key: "cfEmail|bookID"
}

func NewKindleStore(dir string) (*KindleStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	ks := &KindleStore{
		kindleFile:     dir + "/kindle_emails.json",
		requestsFile:   dir + "/book_requests.json",
		historyFile:    dir + "/request_history.json",
		sentFile:       dir + "/sent_books.json",
		kindleEmails:   map[string]string{},
		bookRequests:   map[string]string{},
		requestHistory: map[string]BookRequest{},
		sentBooks:      map[string]time.Time{},
	}
	if err := loadJSON(ks.kindleFile, &ks.kindleEmails); err != nil {
		return nil, err
	}
	if err := loadJSON(ks.requestsFile, &ks.bookRequests); err != nil {
		return nil, err
	}
	if err := loadJSON(ks.historyFile, &ks.requestHistory); err != nil {
		return nil, err
	}
	if err := loadJSON(ks.sentFile, &ks.sentBooks); err != nil {
		return nil, err
	}
	return ks, nil
}

func loadJSON(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func saveJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (ks *KindleStore) GetKindleEmail(cfEmail string) string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.kindleEmails[cfEmail]
}

func (ks *KindleStore) SetKindleEmail(cfEmail, kindleEmail string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.kindleEmails[cfEmail] = kindleEmail
	return saveJSON(ks.kindleFile, ks.kindleEmails)
}

// CountRecentRequests returns how many books this user has requested in the last 72h.
func (ks *KindleStore) CountRecentRequests(cfEmail string) int {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	cutoff := time.Now().Add(-requestPeriod)
	count := 0
	for _, req := range ks.requestHistory {
		if req.CFEmail == cfEmail && req.RequestedAt.After(cutoff) {
			count++
		}
	}
	return count
}

// RecordBookRequest adds the request to both the pending store and the rate-limit history.
// Old history entries (>72h) are pruned on each call.
// Returns an error if the user has already reached the limit.
func (ks *KindleStore) RecordBookRequest(bookID, cfEmail string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-requestPeriod)

	// Prune expired history entries.
	for id, req := range ks.requestHistory {
		if req.RequestedAt.Before(cutoff) {
			delete(ks.requestHistory, id)
		}
	}

	// Count recent requests for this user.
	count := 0
	for _, req := range ks.requestHistory {
		if req.CFEmail == cfEmail && req.RequestedAt.After(cutoff) {
			count++
		}
	}
	limit := requestLimit
	if override, ok := requestLimitOverrides[cfEmail]; ok {
		limit = override
	}
	if count >= limit {
		return fmt.Errorf("rate limit: %d/%d requests used in the last 72h", count, limit)
	}

	ks.requestHistory[bookID] = BookRequest{CFEmail: cfEmail, RequestedAt: now}
	ks.bookRequests[bookID] = cfEmail

	if err := saveJSON(ks.historyFile, ks.requestHistory); err != nil {
		return err
	}
	return saveJSON(ks.requestsFile, ks.bookRequests)
}

// QueueKindleDelivery adds a book to the pending delivery queue without checking
// the rate limit. Used when the book is already in LL (e.g. was downloaded but
// the Kindle delivery wasn't recorded due to hitting the rate limit at request time).
func (ks *KindleStore) QueueKindleDelivery(bookID, cfEmail string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.bookRequests[bookID] = cfEmail
	return saveJSON(ks.requestsFile, ks.bookRequests)
}

func (ks *KindleStore) GetRequestingUser(bookID string) string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.bookRequests[bookID]
}

// AllBookRequests returns a copy of the bookID → cfEmail pending map.
func (ks *KindleStore) AllBookRequests() map[string]string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	cp := make(map[string]string, len(ks.bookRequests))
	for k, v := range ks.bookRequests {
		cp[k] = v
	}
	return cp
}

// DeleteBookRequest removes a fulfilled request from the pending store.
// The history entry is kept so the 72h slot remains used.
func (ks *KindleStore) DeleteBookRequest(bookID string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	delete(ks.bookRequests, bookID)
	return saveJSON(ks.requestsFile, ks.bookRequests)
}

// MarkBookSent records that a book was sent to a user's Kindle.
func (ks *KindleStore) MarkBookSent(cfEmail, bookID string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.sentBooks[cfEmail+"|"+bookID] = time.Now()
	return saveJSON(ks.sentFile, ks.sentBooks)
}

// SentBookIDs returns the set of bookIDs already sent to a given user.
func (ks *KindleStore) SentBookIDs(cfEmail string) map[string]bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	prefix := cfEmail + "|"
	out := map[string]bool{}
	for k := range ks.sentBooks {
		if strings.HasPrefix(k, prefix) {
			out[k[len(prefix):]] = true
		}
	}
	return out
}
