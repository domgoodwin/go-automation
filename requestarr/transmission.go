package requestarr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TransmissionClient struct {
	url       string
	sessionID string
	http      *http.Client
}

func NewTransmissionClient(url string) *TransmissionClient {
	return &TransmissionClient{
		url:  strings.TrimRight(url, "/") + "/transmission/rpc",
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

type rpcRequest struct {
	Method    string         `json:"method"`
	Arguments map[string]any `json:"arguments"`
}

type rpcResponse struct {
	Result    string          `json:"result"`
	Arguments json.RawMessage `json:"arguments"`
}

func (c *TransmissionClient) call(method string, args map[string]any, out any) error {
	body, _ := json.Marshal(rpcRequest{Method: method, Arguments: args})

	req, _ := http.NewRequest("POST", c.url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c.sessionID != "" {
		req.Header.Set("X-Transmission-Session-Id", c.sessionID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Transmission returns 409 with a new session ID when ours is stale/missing
	if resp.StatusCode == 409 {
		c.sessionID = resp.Header.Get("X-Transmission-Session-Id")
		return c.call(method, args, out)
	}

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rpc rpcResponse
	if err := json.Unmarshal(rb, &rpc); err != nil {
		return fmt.Errorf("transmission: %w", err)
	}
	if rpc.Result != "success" {
		return fmt.Errorf("transmission: %s", rpc.Result)
	}
	if out != nil {
		return json.Unmarshal(rpc.Arguments, out)
	}
	return nil
}

type Torrent struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	HashString  string  `json:"hashString"`
	Status      int     `json:"status"`
	PercentDone float64 `json:"percentDone"`
	Eta         int     `json:"eta"`
	RateDown    int64   `json:"rateDownload"`
	DoneDate    int64   `json:"doneDate"`
	TotalSize   int64   `json:"totalSize"`
	RequestedBy string  `json:"-"` // populated from Sonarr/Radarr tags
}

// Status codes
const (
	torrentStopped  = 0
	torrentDownload = 4
	torrentSeed     = 6
)

func (t Torrent) IsDownloading() bool { return t.Status == torrentDownload }
func (t Torrent) IsSeeding() bool     { return t.Status == torrentSeed }

func (t Torrent) RecentlyCompleted() bool {
	if t.DoneDate == 0 {
		return false
	}
	return time.Since(time.Unix(t.DoneDate, 0)) < 24*time.Hour
}

func (t Torrent) ProgressPct() int { return int(t.PercentDone * 100) }

func (t Torrent) ETAString() string {
	if t.Eta < 0 {
		return "∞"
	}
	d := time.Duration(t.Eta) * time.Second
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func (t Torrent) SizeString() string {
	gb := float64(t.TotalSize) / 1e9
	if gb >= 1 {
		return fmt.Sprintf("%.1f GB", gb)
	}
	return fmt.Sprintf("%.0f MB", gb*1000)
}

func (t Torrent) SpeedString() string {
	mb := float64(t.RateDown) / 1e6
	return fmt.Sprintf("%.1f MB/s", mb)
}

type torrentList struct {
	Torrents []Torrent `json:"torrents"`
}

func (c *TransmissionClient) GetActive() (downloading []Torrent, recentDone []Torrent, err error) {
	var result torrentList
	err = c.call("torrent-get", map[string]any{
		"fields": []string{"id", "name", "hashString", "status", "percentDone", "eta", "rateDownload", "doneDate", "totalSize"},
	}, &result)
	if err != nil {
		return
	}
	for _, t := range result.Torrents {
		if t.IsDownloading() {
			downloading = append(downloading, t)
		} else if t.RecentlyCompleted() {
			recentDone = append(recentDone, t)
		}
	}
	return
}
