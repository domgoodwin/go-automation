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
		renderGraph()
	},
}

func renderGraph() {
	fig := generateHistoryFigure()
	offline.ToHtml(fig, "./out.html")
}

func generateHistoryFigure() *grob.Fig {
	f, err := os.Open(dataFile)
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
