package requestarr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

type LazyLibrarianClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewLazyLibrarianClient(baseURL, apiKey string) *LazyLibrarianClient {
	return &LazyLibrarianClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{},
	}
}

// LLSearchBook is returned by findBook (Goodreads/Google Books search) — lowercase JSON fields.
type LLSearchBook struct {
	BookID     string  `json:"bookid"`
	BookName   string  `json:"bookname"`
	AuthorName string  `json:"authorname"`
	BookImg    string  `json:"bookimg"` // full HTTP URL from Goodreads CDN
	BookDesc   string  `json:"bookdesc"`
	BookDate   string  `json:"bookdate"`
	BookRate   float64 `json:"bookrate"`
}

// LLLibraryBook is returned by getAllBooks — PascalCase JSON fields.
type LLLibraryBook struct {
	BookID     string `json:"BookID"`
	BookName   string `json:"BookName"`
	AuthorName string `json:"AuthorName"`
	BookDate   string `json:"BookDate"`
	Status     string `json:"Status"` // Have, Wanted, Open, Skipped, Ignored
}

func (c *LazyLibrarianClient) get(params url.Values, out interface{}) error {
	params.Set("apikey", c.apiKey)
	u := c.baseURL + "/api?" + params.Encode()
	resp, err := c.http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lazylibrarian %s: %d %s", params.Get("cmd"), resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Search searches Goodreads/Google Books for books matching query via LL's findBook API.
func (c *LazyLibrarianClient) Search(query string) ([]LLSearchBook, error) {
	var results []LLSearchBook
	err := c.get(url.Values{"cmd": {"findBook"}, "name": {query}}, &results)
	return results, err
}

// AllBooks returns all books in LL's database.
func (c *LazyLibrarianClient) AllBooks() ([]LLLibraryBook, error) {
	var books []LLLibraryBook
	err := c.get(url.Values{"cmd": {"getAllBooks"}}, &books)
	return books, err
}

// LibraryStatus returns a map of BookID → Status for all books in LL's database.
func (c *LazyLibrarianClient) LibraryStatus() (map[string]string, error) {
	books, err := c.AllBooks()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(books))
	for _, b := range books {
		m[b.BookID] = b.Status
	}
	return m, nil
}

func (c *LazyLibrarianClient) getText(params url.Values) (string, error) {
	params.Set("apikey", c.apiKey)
	u := c.baseURL + "/api?" + params.Encode()
	resp, err := c.http.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(body)), err
}

// QueueBook ensures a book is in LL's database then marks it as Wanted.
// findBook returns Goodreads IDs that LL may not know yet, so addBook is called first.
func (c *LazyLibrarianClient) QueueBook(bookID string) error {
	// addBook is idempotent — safe to call even if already in DB.
	if _, err := c.getText(url.Values{"cmd": {"addBook"}, "id": {bookID}, "wait": {""}}); err != nil {
		return fmt.Errorf("addBook: %w", err)
	}
	text, err := c.getText(url.Values{"cmd": {"queueBook"}, "id": {bookID}})
	if err != nil {
		return err
	}
	if text != "OK" {
		return fmt.Errorf("queueBook: unexpected response: %s", text)
	}
	// Trigger the download search immediately (queueBook only sets status to Wanted).
	if _, err := c.getText(url.Values{"cmd": {"searchBook"}, "id": {bookID}}); err != nil {
		return fmt.Errorf("searchBook: %w", err)
	}
	return nil
}

// FindPendingBook finds a pending requested book either by:
//  1. Matching the folder name against book titles in pendingIDs (for the just-downloaded book)
//  2. Any pending book that now has Status=="Have" in LL
//
// Returns deduplicated results — each BookID appears at most once.
func (c *LazyLibrarianClient) FindPendingBook(bookFolder string, pendingIDs map[string]string) ([]LLLibraryBook, error) {
	var books []LLLibraryBook
	if err := c.get(url.Values{"cmd": {"getAllBooks"}}, &books); err != nil {
		return nil, err
	}

	folderNorm := normTitle(filepath.Base(strings.TrimRight(bookFolder, "/")))

	seen := map[string]bool{}
	var matched []LLLibraryBook
	for _, b := range books {
		if _, ok := pendingIDs[b.BookID]; !ok {
			continue
		}
		if seen[b.BookID] {
			continue
		}
		titleNorm := normTitle(b.BookName)
		byFolder := bookFolder != "" && (strings.Contains(folderNorm, titleNorm) || strings.Contains(titleNorm, folderNorm))
		byStatus := b.Status == "Open" || b.Status == "Have"
		if byFolder || byStatus {
			matched = append(matched, b)
			seen[b.BookID] = true
		}
	}
	return matched, nil
}
