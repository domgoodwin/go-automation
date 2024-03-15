package cmd

import (
	"encoding/csv"
	"os"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/MetalBlueberry/go-plotly/offline"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(graphCmd)
}

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Export data as graphs",
	Run: func(cmd *cobra.Command, args []string) {
		renderGraph(dataFile, "./out.html")
	},
}

func renderGraph(inFile string, outFile string) {
	fig := generateHistoryFigure(inFile)
	offline.ToHtml(fig, outFile)
}

func generateHistoryFigure(inFile string) *grob.Fig {
	f, err := os.Open(inFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}
	var xValues []string
	var yValues []string
	for _, line := range records {
		for i, value := range line {
			if i == 0 {
				xValues = append(xValues, value)
			}
			if i == 1 {
				yValues = append(yValues, value)
			}
		}
	}
	fig := &grob.Fig{
		Data: grob.Traces{
			&grob.Bar{
				Type: grob.TraceTypeBar,
				X:    xValues,
				Y:    yValues,
			},
		},
		Layout: &grob.Layout{
			Title: &grob.LayoutTitle{
				Text: "Solar daily generation rates",
			},
			Xaxis: &grob.LayoutXaxis{
				Title: &grob.LayoutXaxisTitle{
					Text: "Date",
				},
				Rangeslider: &grob.LayoutXaxisRangeslider{
					Autorange: grob.True,
				},
				Rangeselector: &grob.LayoutXaxisRangeselector{
					Buttons: []*RangeStepButton{
						{
							Count:    1,
							Label:    "1m",
							Step:     "month",
							Stepmode: "backward",
						},
						{
							Count:    6,
							Label:    "6m",
							Step:     "month",
							Stepmode: "backward",
						},
						{
							Count:    1,
							Label:    "YTD",
							Step:     "year",
							Stepmode: "todate",
						},
						{
							Count:    1,
							Label:    "1y",
							Step:     "year",
							Stepmode: "backward",
						},
						{
							Step: "all",
						},
					},
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

type RangeStepButton struct {
	Count    int    `json:"count,omitempty"`
	Label    string `json:"label,omitempty"`
	Step     string `json:"step,omitempty"`
	Stepmode string `json:"stepmode,omitempty"`
}
