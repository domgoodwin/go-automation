package homeassistant

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/gorilla/websocket"

	log "github.com/sirupsen/logrus"
)

func (c *Client) InitWebsocket() {
	wsc, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/api/websocket", c.url), nil)
	if err != nil {
		panic(err)
	}
	c.wsc = wsc

	mt, _, _ := c.readMessage()
	c.wsType = mt

	_ = c.sendMessage(fmt.Sprintf(`{"type": "auth","access_token": "%s"}`, os.Getenv("HA_TOKEN")))

	c.readMessage()
}

func (c *Client) readMessage() (int, []byte, error) {
	mt, rsp, err := c.wsc.ReadMessage()
	if err != nil {
		panic(err)
	}
	log.Debug(string(rsp))
	return mt, rsp, err
}

func (c *Client) sendMessage(msg string) error {
	err := c.wsc.WriteMessage(c.wsType, []byte(msg))
	if err != nil {
		panic(err)
	}
	c.counter++
	return err
}

func (c *Client) GetDailyData(day time.Time) *RecorderDailyData {
	start := day.Truncate(time.Hour * 24)
	end := start.Add(24 * time.Hour).Add(-1 * time.Second)

	log.Debug("get daily data: %d %v %v\n", c.counter, start, end)
	msg := fmt.Sprintf(`{"type":"recorder/statistics_during_period","start_time":"%s","end_time":"%s","statistic_ids":["sensor.pv_energy"],"period":"hour","units":{"energy":"kWh"},"types":["change"],"id":%d}`, start.Format(time.RFC3339), end.Format(time.RFC3339), c.counter)
	_ = c.sendMessage(msg)
	_, rsp, _ := c.readMessage()

	data := &RecorderDailyData{
		Day: start,
	}
	err := json.Unmarshal(rsp, data)
	if err != nil {
		panic(err)
	}

	return data
}

type RecorderDailyData struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Result  struct {
		SensorPvEnergy []struct {
			Start  int64   `json:"start"`
			End    int64   `json:"end"`
			Change float64 `json:"change"`
		} `json:"sensor.pv_energy"`
	} `json:"result"`

	Day time.Time
}

func (d *RecorderDailyData) GetHourlyData() ([]time.Time, []float64) {
	var timeValues []time.Time
	var sensorValues []float64
	firstNonZeroIndex := -1
	lastNonZeroIndex := 0
	for i, entry := range d.Result.SensorPvEnergy {
		timeVal := time.UnixMilli(entry.Start)
		timeValues = append(timeValues, timeVal)
		sensorValues = append(sensorValues, entry.Change)
		if entry.Change != 0 && firstNonZeroIndex == -1 {
			firstNonZeroIndex = i
		}
		if entry.Change != 0 {
			lastNonZeroIndex = i
		}
	}

	return timeValues[firstNonZeroIndex-1 : lastNonZeroIndex+2], sensorValues[firstNonZeroIndex-1 : lastNonZeroIndex+2]
}

func (d *RecorderDailyData) Total() float64 {
	var total float64
	for _, sensor := range d.Result.SensorPvEnergy {
		total += sensor.Change
	}
	return total
}

func (d *RecorderDailyData) ToFig() *grob.Fig {
	now := time.Now()
	timeValues, sensorValues := d.GetHourlyData()
	log.Debug(timeValues)
	log.Debug(sensorValues)

	for i, timeVal := range timeValues {
		if timeVal.IsZero() && i != 0 {
			timeValues[i] = timeValues[i-1].Add(time.Hour)
		}
	}

	fig := &grob.Fig{
		Data: grob.Traces{
			&grob.Bar{
				Type: grob.TraceTypeBar,
				X:    timeValues,
				Y:    sensorValues,
			},
		},
		Layout: &grob.Layout{
			Title: &grob.LayoutTitle{
				Text: fmt.Sprintf("%s generation", now.Format(time.DateOnly)),
			},
			Xaxis: &grob.LayoutXaxis{
				Title: &grob.LayoutXaxisTitle{
					Text: "Date",
				},
			},
			Yaxis: &grob.LayoutYaxis{
				Title: &grob.LayoutYaxisTitle{
					Text: "Generated (kWh)",
				},
				Ticksuffix: "kWh",
			},
		},
	}
	return fig
}

func (d *RecorderDailyData) Status() string {
	return fmt.Sprintf("Total generated today: %.2fkWh", d.Total())
}

func (d *RecorderDailyData) IsEmpty() bool {
	return len(d.Result.SensorPvEnergy) == 0
}

func (d *RecorderDailyData) CSVHeader() string {
	return "day,solar_generation"
}

func (d *RecorderDailyData) CSVLine() string {
	return fmt.Sprintf("%v,%v", d.Day.Format(time.DateOnly), d.Total())
}
