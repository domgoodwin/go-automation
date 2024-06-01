package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func (c *Client) sendAPIRequest(ctx context.Context, path string) ([]byte, error) {
	fullUrl := fmt.Sprintf("http://%s%s", c.url, path)
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Add("Content-Type", "application/json")
	log.Debug(req)
	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf(fmt.Sprintf("non 200 code, rsp: %v", string(body)))
	}
	return body, nil
}

func (c *Client) GetHistoryDailyData(ctx context.Context, endTime time.Time) (*HistoryDailyData, error) {
	endTimeStr := endTime.UTC().Format(time.RFC3339)
	startTimeStr := endTime.UTC().Add(time.Hour * -24).Format(time.RFC3339)

	path := fmt.Sprintf("/api/history/period/%s?filter_entity_id=sensor.pv_energy&end_time=%s", startTimeStr, endTimeStr)
	body, err := c.sendAPIRequest(ctx, path)
	if err != nil {
		log.Errorf("failed to send api request %v", err)
		return nil, err
	}
	var historyValues [][]*History
	logrus.Debugf("get history daily data body: %v", string(body))
	err = json.Unmarshal(body, &historyValues)
	if err != nil {
		log.Errorf("failed to unmarshal %v", err)
		return nil, err
	}
	if len(historyValues) == 0 {
		log.Info("nothing returned")
		return nil, nil
	}
	return &HistoryDailyData{
		values:  historyValues[0],
		endTime: endTime.UTC(),
	}, nil
}

type HistoryDailyData struct {
	values  []*History
	endTime time.Time
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
		log.Debug(history)
		if history.State == "unknown" {
			log.Debug("skipping entry (unknown state)")
			continue
		}
		if minTime.IsZero() || history.LastChanged.Before(minTime) {
			if !h.endTime.Truncate(time.Hour * 24).Equal(history.LastChanged.Truncate(time.Hour * 24)) {
				log.Debug("skipping entry (truncate not equal)")
				continue
			}
			log.Debugf("setting min: %v", history.State)
			minTime = history.LastChanged
			min = history.State
		}
		if maxTime.IsZero() || history.LastChanged.After(maxTime) {
			log.Debugf("setting max: %v", history.State)
			maxTime = history.LastChanged
			max = history.State
		}
	}

	h.maxTime = maxTime
	h.minTime = minTime

	log.Debugf("comparing min: %v and max: %v", min, max)
	var minFloat, maxFloat float64
	minFloat, err := strconv.ParseFloat(min, 64)
	if err != nil {
		return err
	}
	maxFloat, err = strconv.ParseFloat(max, 64)
	if err != nil {
		return err
	}
	if minFloat > maxFloat {
		log.Debug("setting min to zero as higher then max")
		min = "0"
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
