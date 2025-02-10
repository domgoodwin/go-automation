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
	// generateEChartExport(inFile, "./out-2.html")
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

// func generateEChartExport(inFile, outFile string) {
// 	fIn, err := os.Open(inFile)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer fIn.Close()

// 	csvReader := csv.NewReader(fIn)
// 	records, err := csvReader.ReadAll()
// 	if err != nil {
// 		panic(err)
// 	}
// 	var xValues []string
// 	var yValues []opts.BarData

// 	var rawData [][]interface{}
// 	for _, line := range records {
// 		entry := make([]interface{}, 2)
// 		for i, value := range line {
// 			if i == 0 {
// 				xTime, err := time.Parse("2006-01-02", value)
// 				if err != nil {
// 					log.Errorf("failed to parse time %v:%v", value, err)
// 					continue
// 				}
// 				// xValues = append(xValues, xTime.Format(time.RFC3339))
// 				entry[0] = xTime.Format(time.RFC3339)
// 			}
// 			if i == 1 {
// 				// yValues = append(yValues, opts.BarData{
// 				// 	Value: value,
// 				// })
// 				entry[1] = value
// 			}
// 		}
// 		rawData = append(rawData, entry)
// 	}
// 	var data []opts.BarData
// 	for _, line := range rawData {
// 		data = append(data, opts.BarData{})
// 	}

// 	bar := charts.NewBar()
// 	bar.SetGlobalOptions(
// 		charts.WithTitleOpts(opts.Title{Title: "Daily Solar Generation"}),
// 		charts.WithYAxisOpts(opts.YAxis{
// 			Show: true,
// 			Name: "Generated",
// 			Type: "value",
// 			Min:  0,
// 			AxisLabel: &opts.AxisLabel{
// 				Show:      true,
// 				Formatter: "{value}kWh",
// 			},
// 		}),
// 		charts.WithXAxisOpts(opts.XAxis{
// 			Name: "Date",
// 			Type: "time",
// 			Show: true,
// 			AxisLabel: &opts.AxisLabel{
// 				Show:      true,
// 				Formatter: "{value}",
// 			},
// 		}),
// 		charts.WithDataZoomOpts(opts.DataZoom{
// 			Type:  "slider",
// 			Start: 80,
// 			End:   100,
// 		}),
// 		charts.WithTooltipOpts(opts.Tooltip{
// 			Show:           true,
// 			Trigger:        "axis",
// 			TriggerOn:      "mousemove",
// 			ValueFormatter: "{value}kWh",
// 		}),
// 		charts.WithLegendOpts(opts.Legend{
// 			Show: true,
// 			Type: "plain",
// 		}),
// 	)

// 	bar.SetXAxis(xValues).
// 		AddSeries("Solar", yValues)

// 	bar.AddSeries()

// 	page := components.NewPage()

// 	page.AddCharts(bar)

// 	f, err := os.Create(outFile)
// 	if err != nil {
// 		log.Errorf("failed to create file: %v", err)
// 		return
// 	}

// 	err = page.Render(io.MultiWriter(f))
// 	if err != nil {
// 		log.Errorf("failed to renger page %v", err)
// 		return
// 	}
// }
