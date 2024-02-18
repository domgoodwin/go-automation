package cmd

import (
	"context"

	"github.com/domgoodwin/go-automation/homeassistant"
	"github.com/spf13/cobra"

	"fmt"
	"os"
	"strconv"
	"time"
)

func init() {
	rootCmd.AddCommand(dailyExportCmd)
}

var dailyExportCmd = &cobra.Command{
	Use:   "daily-export",
	Short: "Export past [arg] day solar figures",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		today := time.Now()
		daysToGoBackStr := args[0]
		daysToGoBack, err := strconv.Atoi(daysToGoBackStr)
		if err != nil {
			panic(err)
		}
		for i := daysToGoBack; i >= 0; i-- {
			err := dailyExport(context.Background(), today.Add(time.Duration(i)*(-24*time.Hour)))
			if err != nil {
				panic(err)
			}
		}
	},
}

const (
	dataFile = "data.csv"
)

func dailyExport(ctx context.Context, endTime time.Time) error {
	c := homeassistant.CreateClient()
	data, err := c.GetHistoryDailyData(ctx, endTime)
	if err != nil {
		return err
	}
	increase, err := data.GetChange()
	if err != nil {
		return err
	}
	dateValue := data.DataDate().Format("2006-01-02")

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(dataFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write([]byte(fmt.Sprintf("%s,%.2f\n", dateValue, increase))); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}
