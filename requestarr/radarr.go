package requestarr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type RadarrClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewRadarrClient(baseURL, apiKey string) *RadarrClient {
	return &RadarrClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{},
	}
}

type RadarrMovie struct {
	TmdbID    int           `json:"tmdbId"`
	Title     string        `json:"title"`
	Year      int           `json:"year"`
	Overview  string        `json:"overview"`
	TitleSlug string        `json:"titleSlug"`
	Images    []RadarrImage `json:"images"`
	ID        int           `json:"id"`   // only set if already in library
	Tags      []int         `json:"tags"` // tag IDs
}

type RadarrImage struct {
	CoverType string `json:"coverType"`
	RemoteURL string `json:"remoteUrl"`
}

type RadarrRootFolder struct {
	Path string `json:"path"`
}

type RadarrQualityProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (c *RadarrClient) get(path string, params url.Values, out interface{}) error {
	u := fmt.Sprintf("%s/api/v3/%s", c.baseURL, path)
	if params == nil {
		params = url.Values{}
	}
	params.Set("apikey", c.apiKey)
	resp, err := c.http.Get(u + "?" + params.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("radarr %s: %d %s", path, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *RadarrClient) Lookup(query string) ([]RadarrMovie, error) {
	var results []RadarrMovie
	err := c.get("movie/lookup", url.Values{"term": {query}}, &results)
	return results, err
}

func (c *RadarrClient) Library() ([]RadarrMovie, error) {
	var results []RadarrMovie
	err := c.get("movie", nil, &results)
	return results, err
}

func (c *RadarrClient) RootFolders() ([]RadarrRootFolder, error) {
	var results []RadarrRootFolder
	err := c.get("rootfolder", nil, &results)
	return results, err
}

func (c *RadarrClient) QualityProfiles() ([]RadarrQualityProfile, error) {
	var results []RadarrQualityProfile
	err := c.get("qualityprofile", nil, &results)
	return results, err
}

type RadarrTag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

func (c *RadarrClient) EnsureTag(label string) (int, error) {
	var tags []RadarrTag
	if err := c.get("tag", nil, &tags); err != nil {
		return 0, err
	}
	for _, t := range tags {
		if t.Label == label {
			return t.ID, nil
		}
	}
	b, _ := json.Marshal(map[string]string{"label": label})
	u := fmt.Sprintf("%s/api/v3/tag?apikey=%s", c.baseURL, c.apiKey)
	resp, err := c.http.Post(u, "application/json", strings.NewReader(string(b)))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var created RadarrTag
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (c *RadarrClient) Add(movie RadarrMovie, rootFolder string, qualityProfileID int, tagIDs []int) error {
	body := map[string]interface{}{
		"tmdbId":           movie.TmdbID,
		"title":            movie.Title,
		"year":             movie.Year,
		"titleSlug":        movie.TitleSlug,
		"images":           movie.Images,
		"monitored":        true,
		"rootFolderPath":   rootFolder,
		"qualityProfileId": qualityProfileID,
		"tags":             tagIDs,
		"addOptions": map[string]interface{}{
			"searchForMovie": true,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	u := fmt.Sprintf("%s/api/v3/movie?apikey=%s", c.baseURL, c.apiKey)
	resp, err := c.http.Post(u, "application/json", strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("radarr add: %d %s", resp.StatusCode, string(rb))
	}
	return nil
}

func (m RadarrMovie) PosterURL() string {
	for _, img := range m.Images {
		if img.CoverType == "poster" {
			return img.RemoteURL
		}
	}
	return ""
}

func (m RadarrMovie) InLibrary() bool {
	return m.ID > 0
}

type RadarrCalendarMovie struct {
	Title           string `json:"title"`
	PhysicalRelease string `json:"physicalRelease"`
	DigitalRelease  string `json:"digitalRelease"`
	InCinemas       string `json:"inCinemas"`
}

func (c *RadarrClient) Calendar(start, end string) ([]RadarrCalendarMovie, error) {
	var results []RadarrCalendarMovie
	err := c.get("calendar", url.Values{"start": {start}, "end": {end}}, &results)
	return results, err
}

type RadarrQueueItem struct {
	MovieID    int    `json:"movieId"`
	DownloadID string `json:"downloadId"`
}

type RadarrQueuePage struct {
	Records []RadarrQueueItem `json:"records"`
}

func (c *RadarrClient) Queue() ([]RadarrQueueItem, error) {
	var page RadarrQueuePage
	err := c.get("queue", url.Values{"pageSize": {"200"}}, &page)
	return page.Records, err
}

func (c *RadarrClient) TagLabels() (map[int]string, error) {
	var tags []RadarrTag
	if err := c.get("tag", nil, &tags); err != nil {
		return nil, err
	}
	m := make(map[int]string, len(tags))
	for _, t := range tags {
		m[t.ID] = t.Label
	}
	return m, nil
}
