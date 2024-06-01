package influx

import (
	"context"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

const (
	server = "http://192.168.0.240:8086"
)

// measuremeent: pv_panels

func Write(ctx context.Context, measurement string, tags map[string]string, fields map[string]interface{}) error {
	// Create a new client using an InfluxDB server base URL and an authentication token
	client := influxdb2.NewClient(server, os.Getenv("INFLUXDB_TOKEN"))

	org := "home"
	bucket := "solar"
	writeAPI := client.WriteAPIBlocking(org, bucket)

	point := write.NewPoint(measurement, tags, fields, time.Now())
	if err := writeAPI.WritePoint(ctx, point); err != nil {
		return err
	}
	return nil
}
