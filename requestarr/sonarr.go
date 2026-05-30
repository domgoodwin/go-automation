package requestarr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type SonarrClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewSonarrClient(baseURL, apiKey string) *SonarrClient {
	return &SonarrClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{},
	}
}

type SonarrStatistics struct {
	EpisodeFileCount int `json:"episodeFileCount"`
}

type SonarrSeries struct {
	TvdbID     int              `json:"tvdbId"`
	Title      string           `json:"title"`
	Year       int              `json:"year"`
	Overview   string           `json:"overview"`
	TitleSlug  string           `json:"titleSlug"`
	Images     []SonarrImage    `json:"images"`
	Seasons    []SonarrSeason   `json:"seasons"`
	Monitored  bool             `json:"monitored"`
	ID         int              `json:"id"`         // only set if already in library
	Tags       []int            `json:"tags"`       // tag IDs
	Statistics SonarrStatistics `json:"statistics"` // only populated from library
}

type SonarrImage struct {
	CoverType string `json:"coverType"`
	RemoteURL string `json:"remoteUrl"`
}

type SonarrSeason struct {
	SeasonNumber int  `json:"seasonNumber"`
	Monitored    bool `json:"monitored"`
}

type SonarrRootFolder struct {
	Path string `json:"path"`
}

type SonarrQualityProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (c *SonarrClient) get(path string, params url.Values, out interface{}) error {
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
		return fmt.Errorf("sonarr %s: %d %s", path, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *SonarrClient) Lookup(query string) ([]SonarrSeries, error) {
	var results []SonarrSeries
	err := c.get("series/lookup", url.Values{"term": {query}}, &results)
	return results, err
}

func (c *SonarrClient) Library() ([]SonarrSeries, error) {
	var results []SonarrSeries
	err := c.get("series", nil, &results)
	return results, err
}

func (c *SonarrClient) RootFolders() ([]SonarrRootFolder, error) {
	var results []SonarrRootFolder
	err := c.get("rootfolder", nil, &results)
	return results, err
}

func (c *SonarrClient) QualityProfiles() ([]SonarrQualityProfile, error) {
	var results []SonarrQualityProfile
	err := c.get("qualityprofile", nil, &results)
	return results, err
}

type SonarrTag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

func (c *SonarrClient) EnsureTag(label string) (int, error) {
	var tags []SonarrTag
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
	var created SonarrTag
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (c *SonarrClient) Add(series SonarrSeries, rootFolder string, qualityProfileID int, tagIDs []int) error {
	series.Monitored = true
	for i := range series.Seasons {
		series.Seasons[i].Monitored = true
	}

	body := map[string]interface{}{
		"tvdbId":           series.TvdbID,
		"title":            series.Title,
		"year":             series.Year,
		"titleSlug":        series.TitleSlug,
		"images":           series.Images,
		"seasons":          series.Seasons,
		"monitored":        true,
		"rootFolderPath":   rootFolder,
		"qualityProfileId": qualityProfileID,
		"tags":             tagIDs,
		"addOptions": map[string]interface{}{
			"searchForMissingEpisodes": true,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	u := fmt.Sprintf("%s/api/v3/series?apikey=%s", c.baseURL, c.apiKey)
	resp, err := c.http.Post(u, "application/json", strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sonarr add: %d %s", resp.StatusCode, string(rb))
	}
	return nil
}

func (s SonarrSeries) PosterURL() string {
	for _, img := range s.Images {
		if img.CoverType == "poster" {
			return img.RemoteURL
		}
	}
	return ""
}

func (s SonarrSeries) InLibrary() bool {
	return s.ID > 0
}

type SonarrEpisode struct {
	Title      string `json:"title"`
	AirDateUtc string `json:"airDateUtc"`
	SeasonNum  int    `json:"seasonNumber"`
	EpisodeNum int    `json:"episodeNumber"`
	Series     struct {
		Title string `json:"title"`
	} `json:"series"`
}

func (c *SonarrClient) Calendar(start, end string) ([]SonarrEpisode, error) {
	var results []SonarrEpisode
	err := c.get("calendar", url.Values{"start": {start}, "end": {end}, "includeSeries": {"true"}}, &results)
	return results, err
}

type SonarrQueueItem struct {
	SeriesID   int    `json:"seriesId"`
	DownloadID string `json:"downloadId"`
}

type SonarrQueuePage struct {
	Records []SonarrQueueItem `json:"records"`
}

func (c *SonarrClient) Queue() ([]SonarrQueueItem, error) {
	var page SonarrQueuePage
	err := c.get("queue", url.Values{"pageSize": {"200"}}, &page)
	return page.Records, err
}

func (c *SonarrClient) TagLabels() (map[int]string, error) {
	var tags []SonarrTag
	if err := c.get("tag", nil, &tags); err != nil {
		return nil, err
	}
	m := make(map[int]string, len(tags))
	for _, t := range tags {
		m[t.ID] = t.Label
	}
	return m, nil
}
