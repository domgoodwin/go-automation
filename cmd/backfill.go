package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/domgoodwin/go-automation/homeassistant"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(backfillCmd)
}

const (
	backfillFile = "backfill.csv"
)

var backfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "Export backfill figures",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		daysToGoBackStr := args[0]
		daysToGoBack, err := strconv.Atoi(daysToGoBackStr)
		if err != nil {
			panic(err)
		}
		backfill(daysToGoBack)
		renderGraph(backfillFile, "./backfill.html")
	},
}

func backfill(count int) {
	ctx := context.Background()

	data := getBackfillData(ctx, count)
	if data == nil {
		fmt.Println("empty data")
		return
	}

	csvContents := ""
	for _, entry := range data {
		csvContents += fmt.Sprintf("%v\n", entry.CSVLine())
	}

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(backfillFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	if _, err := f.Write([]byte(csvContents)); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}

}

func getBackfillData(ctx context.Context, days int) []*homeassistant.RecorderDailyData {
	start := time.Now().Add(-1 * time.Duration(days) * (time.Hour * 24))
	c := homeassistant.CreateClient()
	c.InitWebsocket()
	var dailyDatas []*homeassistant.RecorderDailyData

	for i := 0; i < days; i++ {
		dailyData := c.GetDailyData(start)
		start = start.Add(time.Hour * 24)
		if dailyData.IsEmpty() {
			continue
		}
		dailyDatas = append(dailyDatas, dailyData)

	}

	return dailyDatas
}
