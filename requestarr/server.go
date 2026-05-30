package requestarr

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]`)
var trailingYear = regexp.MustCompile(`\s*[\(\[]\d{4}[\)\]]\s*$`)

func normTitle(s string) string {
	return nonAlphanumeric.ReplaceAllString(strings.ToLower(s), "")
}

func normFolderTitle(s string) string {
	return normTitle(trailingYear.ReplaceAllString(s, ""))
}

func existsOnDisk(rootPath, title string) bool {
	if rootPath == "" {
		return false
	}
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return false
	}
	needle := normTitle(title)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		folder := normFolderTitle(e.Name())
		if strings.Contains(folder, needle) || strings.Contains(needle, folder) {
			return true
		}
	}
	return false
}

func yearFromDate(s string) int {
	if len(s) >= 4 {
		if y, err := strconv.Atoi(s[:4]); err == nil {
			return y
		}
	}
	return 0
}

//go:embed templates/*
var templateFS embed.FS

type Server struct {
	sonarr              *SonarrClient
	radarr              *RadarrClient
	transmission        *TransmissionClient
	ll                  *LazyLibrarianClient // nil if not configured
	store               *KindleStore         // nil if not configured
	smtp                *SMTPSender          // nil if not configured
	tvPath              string
	moviePath           string
	animePath           string
	booksPath           string
	animeQualityProfile string
	smtpFrom            string
	templates           *template.Template
}

type ServerConfig struct {
	SonarrURL, SonarrKey        string
	RadarrURL, RadarrKey        string
	TransmissionURL             string
	TVPath, MoviePath           string
	AnimePath, AnimeQualityProfile string
	LLURL, LLKey                string
	BooksPath                   string
	KindleStorePath             string
	SMTPHost, SMTPPort          string
	SMTPUser, SMTPPass          string
}

func NewServer(cfg ServerConfig) (*Server, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	srv := &Server{
		sonarr:              NewSonarrClient(cfg.SonarrURL, cfg.SonarrKey),
		radarr:              NewRadarrClient(cfg.RadarrURL, cfg.RadarrKey),
		transmission:        NewTransmissionClient(cfg.TransmissionURL),
		tvPath:              cfg.TVPath,
		moviePath:           cfg.MoviePath,
		animePath:           cfg.AnimePath,
		booksPath:           cfg.BooksPath,
		animeQualityProfile: cfg.AnimeQualityProfile,
		smtpFrom:            cfg.SMTPUser,
		templates:           tmpl,
	}

	if cfg.LLURL != "" && cfg.LLKey != "" {
		srv.ll = NewLazyLibrarianClient(cfg.LLURL, cfg.LLKey)
	}

	if cfg.KindleStorePath != "" {
		store, err := NewKindleStore(cfg.KindleStorePath)
		if err != nil {
			return nil, fmt.Errorf("kindle store: %w", err)
		}
		srv.store = store
	}

	if cfg.SMTPHost != "" && cfg.SMTPUser != "" && cfg.SMTPPass != "" {
		sender, err := NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)
		if err != nil {
			return nil, fmt.Errorf("smtp: %w", err)
		}
		srv.smtp = sender
	}

	return srv, nil
}

func pickSonarrProfile(profiles []SonarrQualityProfile, prefer string) int {
	for _, p := range profiles {
		if strings.EqualFold(p.Name, prefer) {
			return p.ID
		}
	}
	return profiles[0].ID
}

func pickRadarrProfile(profiles []RadarrQualityProfile, prefer string) int {
	for _, p := range profiles {
		if strings.EqualFold(p.Name, prefer) {
			return p.ID
		}
	}
	return profiles[0].ID
}

func (s *Server) Start(addr string) error {
	if s.ll != nil && s.store != nil && s.smtp != nil {
		go s.pollPendingBooks()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /search", s.handleSearch)
	mux.HandleFunc("POST /request", s.handleRequest)
	mux.HandleFunc("GET /downloads", s.handleDownloads)
	mux.HandleFunc("GET /calendar", s.handleCalendar)
	mux.HandleFunc("GET /settings", s.handleSettingsGet)
	mux.HandleFunc("POST /settings", s.handleSettingsPost)
	mux.HandleFunc("POST /ll-webhook", s.handleLLWebhook)
	mux.HandleFunc("POST /send", s.handleSend)
	log.Infof("requestarr listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

// pollPendingBooks checks every 5 minutes whether any pending book requests have
// been downloaded by LL and, if so, emails them to the requester's Kindle.
func (s *Server) pollPendingBooks() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.deliverPendingBooks()
	}
}

func (s *Server) deliverPendingBooks() {
	pending := s.store.AllBookRequests()
	if len(pending) == 0 {
		return
	}

	allBooks, err := s.ll.AllBooks()
	if err != nil {
		log.Warnf("book-poller: could not fetch LL library: %v", err)
		return
	}

	for _, book := range allBooks {
		cfEmail, ok := pending[book.BookID]
		if !ok {
			continue
		}
		if book.Status != "Open" && book.Status != "Have" {
			continue
		}

		kindleEmail := s.store.GetKindleEmail(cfEmail)
		if kindleEmail == "" {
			log.Infof("book-poller: requester %s has no Kindle email, skipping %s", cfEmail, book.BookName)
			s.store.DeleteBookRequest(book.BookID)
			continue
		}

		bookFile, err := findBookOnDisk(s.booksPath, book.AuthorName, book.BookName)
		if err != nil {
			log.Warnf("book-poller: %v (will retry next poll)", err)
			continue
		}

		if err := s.smtp.SendBook(kindleEmail, bookFile, book.BookName, book.AuthorName); err != nil {
			log.Errorf("book-poller: SMTP send to %s failed for %s: %v", kindleEmail, book.BookName, err)
			continue
		}

		log.Infof("book-poller: sent %q to %s (Kindle: %s)", book.BookName, cfEmail, kindleEmail)
		if err := s.store.MarkBookSent(cfEmail, book.BookID); err != nil {
			log.Warnf("book-poller: mark sent: %v", err)
		}
		s.store.DeleteBookRequest(book.BookID)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := map[string]bool{"BooksEnabled": s.ll != nil}
	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type SearchResult struct {
	ID        string
	Title     string
	Author    string // books only
	Year      int
	Overview  string
	PosterURL string
	InLibrary bool
	OnDisk    bool
	Searching bool
	Monitored bool
	Sent            bool   // book already sent to this user's Kindle
	CanQueueKindle  bool   // in LL but missing from pending queue (e.g. rate-limited at request time)
	Type            string // "tv", "movie", "anime", "book"
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	mediaType := r.URL.Query().Get("type")
	if mediaType == "" {
		mediaType = "tv"
	}

	if q == "" {
		s.templates.ExecuteTemplate(w, "results.html", nil)
		return
	}

	var results []SearchResult
	var searchErr error

	switch mediaType {
	case "tv", "anime":
		library, err := s.sonarr.Library()
		if err != nil {
			log.Warnf("sonarr library fetch failed: %v", err)
		}
		inLibrary := map[int]bool{}
		sonarrID := map[int]int{}
		hasFiles := map[int]bool{}
		for _, s := range library {
			inLibrary[s.TvdbID] = true
			sonarrID[s.TvdbID] = s.ID
			hasFiles[s.TvdbID] = s.Statistics.EpisodeFileCount > 0
		}
		queuedSeries := map[int]bool{}
		if queue, err := s.sonarr.Queue(); err == nil {
			for _, item := range queue {
				queuedSeries[item.SeriesID] = true
			}
		} else {
			log.Warnf("sonarr queue fetch failed: %v", err)
		}

		diskPath := s.tvPath
		if mediaType == "anime" {
			diskPath = s.animePath
		}

		shows, err := s.sonarr.Lookup(q)
		if err != nil {
			searchErr = err
		}
		for _, show := range shows {
			inLib := inLibrary[show.TvdbID] || show.InLibrary()
			sid := sonarrID[show.TvdbID]
			if sid == 0 {
				sid = show.ID
			}
			searching := inLib && queuedSeries[sid]
			monitored := inLib && !searching && !hasFiles[show.TvdbID]
			results = append(results, SearchResult{
				ID:        fmt.Sprintf("%d", show.TvdbID),
				Title:     show.Title,
				Year:      show.Year,
				Overview:  show.Overview,
				PosterURL: show.PosterURL(),
				InLibrary: inLib && !monitored,
				OnDisk:    existsOnDisk(diskPath, show.Title),
				Searching: searching,
				Monitored: monitored,
				Type:      mediaType,
			})
		}

	case "movie":
		library, err := s.radarr.Library()
		if err != nil {
			log.Warnf("radarr library fetch failed: %v", err)
		}
		inLibrary := map[int]bool{}
		radarrID := map[int]int{}
		for _, m := range library {
			inLibrary[m.TmdbID] = true
			radarrID[m.TmdbID] = m.ID
		}
		queuedMovies := map[int]bool{}
		if queue, err := s.radarr.Queue(); err == nil {
			for _, item := range queue {
				queuedMovies[item.MovieID] = true
			}
		} else {
			log.Warnf("radarr queue fetch failed: %v", err)
		}

		movies, err := s.radarr.Lookup(q)
		if err != nil {
			searchErr = err
		}
		for _, movie := range movies {
			inLib := inLibrary[movie.TmdbID] || movie.InLibrary()
			rid := radarrID[movie.TmdbID]
			if rid == 0 {
				rid = movie.ID
			}
			results = append(results, SearchResult{
				ID:        fmt.Sprintf("%d", movie.TmdbID),
				Title:     movie.Title,
				Year:      movie.Year,
				Overview:  movie.Overview,
				PosterURL: movie.PosterURL(),
				InLibrary: inLib,
				OnDisk:    existsOnDisk(s.moviePath, movie.Title),
				Searching: inLib && queuedMovies[rid],
				Type:      "movie",
			})
		}

	case "book":
		if s.ll == nil {
			http.Error(w, "Lazy Librarian not configured", http.StatusServiceUnavailable)
			return
		}
		books, err := s.ll.Search(q)
		if err != nil {
			searchErr = err
		}
		// Enrich with library status, pending requests, and sent history.
		libStatus, _ := s.ll.LibraryStatus()
		var sentIDs map[string]bool
		var pendingIDs map[string]string // bookID → cfEmail
		if s.store != nil {
			cfEmail := r.Header.Get("Cf-Access-Authenticated-User-Email")
			sentIDs = s.store.SentBookIDs(cfEmail)
			pendingIDs = s.store.AllBookRequests()
		}
		for _, b := range books {
			status := libStatus[b.BookID]
			inLib := status == "Open" || status == "Have"
			inLL := inLib || status == "Snatched"
			_, pendingSend := pendingIDs[b.BookID]
			alreadySent := inLib && sentIDs[b.BookID]
			results = append(results, SearchResult{
				ID:             b.BookID,
				Title:          b.BookName,
				Author:         b.AuthorName,
				Year:           yearFromDate(b.BookDate),
				Overview:       b.BookDesc,
				PosterURL:      b.BookImg,
				InLibrary:      inLib && !pendingSend,
				Searching:      status == "Snatched" || pendingSend,
				Monitored:      status == "Wanted",
				Sent:           alreadySent,
				CanQueueKindle: inLL && !pendingSend && !alreadySent,
				Type:           "book",
			})
		}
	}

	if searchErr != nil {
		http.Error(w, searchErr.Error(), http.StatusBadGateway)
		return
	}

	if mediaType == "book" && s.store != nil {
		cfEmail := r.Header.Get("Cf-Access-Authenticated-User-Email")
		used := s.store.CountRecentRequests(cfEmail)
		limit := RequestLimit(cfEmail)
		color := "#6fbf6f"
		if used >= limit {
			color = "#bf6f6f"
		} else if used >= limit-1 {
			color = "#c8903a"
		}
		fmt.Fprintf(w, `<p style="font-size:0.8rem;color:%s;margin-bottom:0.75rem">%d / %d book requests used in the last 72h</p>`, color, used, limit)
	}

	s.templates.ExecuteTemplate(w, "results.html", results)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cfEmail := r.Header.Get("Cf-Access-Authenticated-User-Email")

	if body.Type == "book" {
		if s.ll == nil {
			http.Error(w, "Lazy Librarian not configured", http.StatusServiceUnavailable)
			return
		}
		// Check if already in LL — if so skip QueueBook (idempotent).
		alreadyInLL := false
		if libStatus, err := s.ll.LibraryStatus(); err == nil {
			st := libStatus[body.ID]
			alreadyInLL = st == "Snatched" || st == "Open" || st == "Have"
		}
		if !alreadyInLL {
			if err := s.ll.QueueBook(body.ID); err != nil {
				log.Errorf("queueBook failed: %v", err)
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		} else {
			log.Infof("request: book %s already in LL, skipping QueueBook", body.ID)
		}
		if s.store != nil && cfEmail != "" {
			if err := s.store.RecordBookRequest(body.ID, cfEmail); err != nil {
				log.Warnf("book request rejected: %v", err)
				w.Write([]byte(fmt.Sprintf(`<span class="badge" style="background:#3a1a1a;color:#bf6f6f">%s</span>`, err.Error())))
				return
			}
		}
		w.Write([]byte(`<span class="badge added">Requested!</span>`))
		return
	}

	// Tag with the requesting user's email (set by Cloudflare Access).
	var tagIDs []int
	if cfEmail != "" {
		if body.Type == "tv" || body.Type == "anime" {
			if id, err := s.sonarr.EnsureTag(cfEmail); err == nil {
				tagIDs = []int{id}
			} else {
				log.Warnf("sonarr ensure tag: %v", err)
			}
		} else {
			if id, err := s.radarr.EnsureTag(cfEmail); err == nil {
				tagIDs = []int{id}
			} else {
				log.Warnf("radarr ensure tag: %v", err)
			}
		}
	}

	var addErr error

	switch body.Type {
	case "tv", "anime":
		shows, err := s.sonarr.Lookup("tvdb:" + body.ID)
		if err != nil || len(shows) == 0 {
			http.Error(w, "show not found", http.StatusNotFound)
			return
		}
		var match *SonarrSeries
		for _, show := range shows {
			if fmt.Sprintf("%d", show.TvdbID) == body.ID {
				match = &show
				break
			}
		}
		if match == nil {
			match = &shows[0]
		}

		rootFolders, err := s.sonarr.RootFolders()
		if err != nil || len(rootFolders) == 0 {
			http.Error(w, "could not get sonarr root folders", http.StatusBadGateway)
			return
		}
		profiles, err := s.sonarr.QualityProfiles()
		if err != nil || len(profiles) == 0 {
			http.Error(w, "could not get sonarr quality profiles", http.StatusBadGateway)
			return
		}

		rootPath := rootFolders[0].Path
		profileName := "Any"

		if body.Type == "anime" && s.animePath != "" {
			for _, rf := range rootFolders {
				if rf.Path == s.animePath || strings.HasPrefix(rf.Path, s.animePath) {
					rootPath = rf.Path
					break
				}
			}
			if s.animeQualityProfile != "" {
				profileName = s.animeQualityProfile
			}
		}

		addErr = s.sonarr.Add(*match, rootPath, pickSonarrProfile(profiles, profileName), tagIDs)

	case "movie":
		movies, err := s.radarr.Lookup("tmdb:" + body.ID)
		if err != nil || len(movies) == 0 {
			http.Error(w, "movie not found", http.StatusNotFound)
			return
		}
		var match *RadarrMovie
		for _, m := range movies {
			if fmt.Sprintf("%d", m.TmdbID) == body.ID {
				match = &m
				break
			}
		}
		if match == nil {
			match = &movies[0]
		}

		rootFolders, err := s.radarr.RootFolders()
		if err != nil || len(rootFolders) == 0 {
			http.Error(w, "could not get radarr root folders", http.StatusBadGateway)
			return
		}
		profiles, err := s.radarr.QualityProfiles()
		if err != nil || len(profiles) == 0 {
			http.Error(w, "could not get radarr quality profiles", http.StatusBadGateway)
			return
		}

		addErr = s.radarr.Add(*match, rootFolders[0].Path, pickRadarrProfile(profiles, "Any"), tagIDs)
	}

	if addErr != nil {
		log.Errorf("add failed: %v", addErr)
		if strings.Contains(addErr.Error(), "already") || strings.Contains(addErr.Error(), "exists") {
			w.Write([]byte(`<span class="badge already">Already in library</span>`))
			return
		}
		http.Error(w, addErr.Error(), http.StatusBadGateway)
		return
	}

	w.Write([]byte(`<span class="badge added">Requested!</span>`))
}

// findBookOnDisk searches booksRoot/<author>/ for a directory matching title and returns
// the first book file found inside it.
func findBookOnDisk(booksRoot, author, title string) (string, error) {
	authorDir := filepath.Join(booksRoot, author)
	entries, err := os.ReadDir(authorDir)
	if err != nil {
		return "", fmt.Errorf("author dir %q not found: %w", authorDir, err)
	}
	titleNorm := normFolderTitle(title)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		folderNorm := normFolderTitle(e.Name())
		if strings.Contains(folderNorm, titleNorm) || strings.Contains(titleNorm, folderNorm) {
			if f, err := findBookFile(filepath.Join(authorDir, e.Name())); err == nil {
				return f, nil
			}
		}
	}
	return "", fmt.Errorf("no book file found for %q by %q in %q", title, author, booksRoot)
}

// handleSend sends an already-downloaded book to the user's Kindle without counting
// against the request rate limit (the book is already in the library).
func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if s.ll == nil || s.store == nil || s.smtp == nil {
		http.Error(w, "book delivery not configured", http.StatusServiceUnavailable)
		return
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cfEmail := r.Header.Get("Cf-Access-Authenticated-User-Email")
	kindleEmail := s.store.GetKindleEmail(cfEmail)
	if kindleEmail == "" {
		w.Write([]byte(`<span class="badge" style="background:#3a2a1a;color:#c8903a">Set your Kindle email in Settings first</span>`))
		return
	}

	// Find the book in LL's library.
	allBooks, err := s.ll.AllBooks()
	if err != nil {
		http.Error(w, "could not query LL", http.StatusBadGateway)
		return
	}
	var book *LLLibraryBook
	for i, b := range allBooks {
		if b.BookID == body.ID {
			book = &allBooks[i]
			break
		}
	}
	if book == nil || (book.Status != "Open" && book.Status != "Have") {
		w.Write([]byte(`<span class="badge" style="background:#3a1a1a;color:#bf6f6f">Book not in library</span>`))
		return
	}

	if s.booksPath == "" {
		http.Error(w, "BOOKS_PATH not configured", http.StatusInternalServerError)
		return
	}

	bookFile, err := findBookOnDisk(s.booksPath, book.AuthorName, book.BookName)
	if err != nil {
		log.Warnf("send: %v", err)
		w.Write([]byte(`<span class="badge" style="background:#3a1a1a;color:#bf6f6f">File not found on disk</span>`))
		return
	}

	if err := s.smtp.SendBook(kindleEmail, bookFile, book.BookName, book.AuthorName); err != nil {
		log.Errorf("send: SMTP failed: %v", err)
		http.Error(w, "failed to send", http.StatusInternalServerError)
		return
	}

	log.Infof("send: sent %q to %s (Kindle: %s)", book.BookName, cfEmail, kindleEmail)
	if err := s.store.MarkBookSent(cfEmail, body.ID); err != nil {
		log.Warnf("send: mark sent: %v", err)
	}
	w.Write([]byte(`<span class="badge already">Sent to Kindle!</span>`))
}


// handleLLWebhook is called by the LL post-processing script after a book download.
// Script should POST: {"bookType":"ebook","bookFolder":"/path/to/folder"}
//
// Rather than correlating by folder path (fragile), we check all pending requests
// against LL's library — any that are now "Have" get delivered and cleared.
func (s *Server) handleLLWebhook(w http.ResponseWriter, r *http.Request) {
	if s.ll == nil || s.store == nil || s.smtp == nil {
		http.Error(w, "book delivery not configured", http.StatusServiceUnavailable)
		return
	}

	var body struct {
		BookType   string `json:"bookType"`
		BookFolder string `json:"bookFolder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Get all pending book requests and check which are now downloaded in LL.
	pending := s.store.AllBookRequests()
	if len(pending) == 0 {
		w.Write([]byte("ok (no pending requests)"))
		return
	}

	downloaded, err := s.ll.FindPendingBook(body.BookFolder, pending)
	if err != nil {
		log.Warnf("ll-webhook: could not fetch downloaded books from LL: %v", err)
		http.Error(w, "could not query LL library", http.StatusBadGateway)
		return
	}

	if len(downloaded) == 0 {
		log.Infof("ll-webhook: triggered for %q but no pending requested books are downloaded yet", body.BookFolder)
		w.Write([]byte("ok (none ready)"))
		return
	}

	for _, book := range downloaded {
		cfEmail := pending[book.BookID]
		kindleEmail := s.store.GetKindleEmail(cfEmail)
		if kindleEmail == "" {
			log.Infof("ll-webhook: requester %s has no Kindle email, skipping %s", cfEmail, book.BookName)
			// Still clear the request so we don't retry indefinitely.
			s.store.DeleteBookRequest(book.BookID)
			continue
		}

		// Find the book file. LL folder structure is typically:
		// <library>/<Author>/<Book Title>/file.epub
		// We search the bookFolder first, then fall back to looking in the LL library.
		bookFile := ""
		if body.BookFolder != "" {
			if f, err := findBookFile(body.BookFolder); err == nil {
				bookFile = f
			}
		}
		if bookFile == "" {
			log.Warnf("ll-webhook: could not find book file for %s in %q, skipping send", book.BookName, body.BookFolder)
			continue
		}

		if err := s.smtp.SendBook(kindleEmail, bookFile, book.BookName, book.AuthorName); err != nil {
			log.Errorf("ll-webhook: SMTP send to %s failed for %s: %v", kindleEmail, book.BookName, err)
			continue
		}

		log.Infof("ll-webhook: sent %q to %s (Kindle: %s)", book.BookName, cfEmail, kindleEmail)
		s.store.DeleteBookRequest(book.BookID)
	}

	w.Write([]byte("ok"))
}

var bookExtensions = map[string]bool{
	".epub": true,
	".mobi": true,
	".azw":  true,
	".azw3": true,
	".pdf":  true,
}

func findBookFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if bookExtensions[strings.ToLower(filepath.Ext(e.Name()))] {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no book file found in %q", dir)
}

type settingsData struct {
	CFEmail     string
	KindleEmail string
	FromEmail   string
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	cfEmail := r.Header.Get("Cf-Access-Authenticated-User-Email")
	var kindleEmail string
	if s.store != nil && cfEmail != "" {
		kindleEmail = s.store.GetKindleEmail(cfEmail)
	}
	data := settingsData{
		CFEmail:     cfEmail,
		KindleEmail: kindleEmail,
		FromEmail:   s.smtpFrom,
	}
	if err := s.templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSettingsPost(w http.ResponseWriter, r *http.Request) {
	cfEmail := r.Header.Get("Cf-Access-Authenticated-User-Email")
	if cfEmail == "" {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	if s.store == nil {
		http.Error(w, "kindle store not configured", http.StatusServiceUnavailable)
		return
	}
	kindleEmail := strings.TrimSpace(r.FormValue("kindle_email"))
	if err := s.store.SetKindleEmail(cfEmail, kindleEmail); err != nil {
		log.Errorf("save kindle email: %v", err)
		w.Write([]byte(`<span class="badge" style="background:#3a1a1a;color:#bf6f6f">Save failed</span>`))
		return
	}
	w.Write([]byte(`<span class="badge already">Saved!</span>`))
}

type downloadsData struct {
	Downloading []Torrent
	RecentDone  []Torrent
}

type CalendarEntry struct {
	Title    string
	Subtitle string
	Type     string // "tv" or "movie"
}

type CalendarDay struct {
	Label   string
	Date    string
	Entries []CalendarEntry
}

func (s *Server) handleCalendar(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	start := now.Format("2006-01-02")
	end := now.AddDate(0, 0, 7).Format("2006-01-02")

	days := map[string]*CalendarDay{}
	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		label := d.Format("Monday")
		if i == 0 {
			label = "Today"
		} else if i == 1 {
			label = "Tomorrow"
		}
		days[key] = &CalendarDay{Label: label, Date: key}
	}

	episodes, err := s.sonarr.Calendar(start, end)
	if err != nil {
		log.Warnf("sonarr calendar error: %v", err)
	}
	for _, ep := range episodes {
		key := ep.AirDateUtc[:10]
		if day, ok := days[key]; ok {
			subtitle := fmt.Sprintf("S%02dE%02d", ep.SeasonNum, ep.EpisodeNum)
			if ep.Title != "" {
				subtitle += " · " + ep.Title
			}
			day.Entries = append(day.Entries, CalendarEntry{
				Title:    ep.Series.Title,
				Subtitle: subtitle,
				Type:     "tv",
			})
		}
	}

	movies, err := s.radarr.Calendar(start, end)
	if err != nil {
		log.Warnf("radarr calendar error: %v", err)
	}
	for _, m := range movies {
		for _, dateStr := range []string{m.DigitalRelease, m.PhysicalRelease, m.InCinemas} {
			if len(dateStr) >= 10 {
				key := dateStr[:10]
				if day, ok := days[key]; ok {
					day.Entries = append(day.Entries, CalendarEntry{
						Title: m.Title,
						Type:  "movie",
					})
					break
				}
			}
		}
	}

	ordered := make([]CalendarDay, 7)
	for i := 0; i < 7; i++ {
		key := now.AddDate(0, 0, i).Format("2006-01-02")
		ordered[i] = *days[key]
	}

	s.templates.ExecuteTemplate(w, "calendar.html", ordered)
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	if s.transmission == nil {
		w.Write([]byte(`<p class="empty">Transmission not configured</p>`))
		return
	}
	downloading, recentDone, err := s.transmission.GetActive()
	if err != nil {
		log.Warnf("transmission error: %v", err)
		w.Write([]byte(`<p class="empty">Could not reach Transmission</p>`))
		return
	}

	hashRequester := map[string]string{}

	sonarrTagLabels, _ := s.sonarr.TagLabels()
	radarrTagLabels, _ := s.radarr.TagLabels()

	if sonarrLib, err := s.sonarr.Library(); err == nil {
		seriesTags := map[int][]int{}
		for _, series := range sonarrLib {
			seriesTags[series.ID] = series.Tags
		}
		if queue, err := s.sonarr.Queue(); err == nil {
			for _, item := range queue {
				hash := strings.ToLower(item.DownloadID)
				for _, tagID := range seriesTags[item.SeriesID] {
					if label, ok := sonarrTagLabels[tagID]; ok {
						hashRequester[hash] = label
						break
					}
				}
			}
		}
	}

	if radarrLib, err := s.radarr.Library(); err == nil {
		movieTags := map[int][]int{}
		for _, movie := range radarrLib {
			movieTags[movie.ID] = movie.Tags
		}
		if queue, err := s.radarr.Queue(); err == nil {
			for _, item := range queue {
				hash := strings.ToLower(item.DownloadID)
				for _, tagID := range movieTags[item.MovieID] {
					if label, ok := radarrTagLabels[tagID]; ok {
						hashRequester[hash] = label
						break
					}
				}
			}
		}
	}

	for i := range downloading {
		downloading[i].RequestedBy = hashRequester[strings.ToLower(downloading[i].HashString)]
	}
	for i := range recentDone {
		recentDone[i].RequestedBy = hashRequester[strings.ToLower(recentDone[i].HashString)]
	}

	s.templates.ExecuteTemplate(w, "downloads.html", downloadsData{
		Downloading: downloading,
		RecentDone:  recentDone,
	})
}
