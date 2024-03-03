package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

func (c *Client) sendAPIRequest(ctx context.Context, path string) ([]byte, error) {
	fullUrl := fmt.Sprintf("http://%s%s", c.url, path)
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Add("Content-Type", "application/json")
	fmt.Println(req)
	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	return io.ReadAll(rsp.Body)
}

func (c *Client) GetHistoryDailyData(ctx context.Context, endTime time.Time) (*HistoryDailyData, error) {
	endTimeStr := endTime.Format(time.RFC3339)
	startTimeStr := endTime.Add(time.Hour * -24).Format(time.RFC3339)

	path := fmt.Sprintf("/api/history/period/%s?filter_entity_id=sensor.pv_energy&end_time=%s", startTimeStr, endTimeStr)
	body, err := c.sendAPIRequest(ctx, path)

	if err != nil {
		return nil, err
	}
	var historyValues [][]*History
	err = json.Unmarshal(body, &historyValues)
	if err != nil {
		return nil, err
	}
	if len(historyValues) == 0 {
		fmt.Println("nothing returned")
		return nil, nil
	}
	return &HistoryDailyData{
		values: historyValues[0],
	}, nil
}

type HistoryDailyData struct {
	values  []*History
	minTime time.Time
	maxTime time.Time
	min     float64
	max     float64
}

type History struct {
	EntityID   string `json:"entity_id"`
	State      string `json:"state"`
	Attributes struct {
		StateClass        string `json:"state_class"`
		UnitOfMeasurement string `json:"unit_of_measurement"`
		DeviceClass       string `json:"device_class"`
		FriendlyName      string `json:"friendly_name"`
	} `json:"attributes"`
	LastChanged time.Time `json:"last_changed"`
	LastUpdated time.Time `json:"last_updated"`
}

func (h *HistoryDailyData) getMinMax() error {
	if !h.minTime.IsZero() || !h.maxTime.IsZero() {
		return nil
	}
	var min, max string
	var minTime, maxTime time.Time
	for _, history := range h.values {
		fmt.Println(history)
		if minTime.IsZero() || history.LastChanged.Before(minTime) {
			minTime = history.LastChanged
			min = history.State
		}
		if maxTime.IsZero() || history.LastChanged.After(maxTime) {
			maxTime = history.LastChanged
			max = history.State
		}
	}

	h.maxTime = maxTime
	h.minTime = minTime

	if min > max {
		fmt.Println("setting min to zero as higher then max")
		min = "0"
	}

	var minFloat, maxFloat float64
	minFloat, err := strconv.ParseFloat(min, 64)
	if err != nil {
		return err
	}
	maxFloat, err = strconv.ParseFloat(max, 64)
	if err != nil {
		return err
	}
	h.min = minFloat
	h.max = maxFloat
	return nil
}

func (h *HistoryDailyData) GetChange() (float64, error) {
	err := h.getMinMax()
	if err != nil {
		return 0, err
	}
	return h.max - h.min, nil
}

func (h HistoryDailyData) DataDate() time.Time {
	return h.maxTime.Truncate(time.Hour * 24)
}
