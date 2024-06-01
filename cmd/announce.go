package cmd

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"time"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/domgoodwin/go-automation/homeassistant"
	"github.com/domgoodwin/go-automation/mastodon"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	jsonFile  = "graph.json"
	imageFile = "out.png"
)

func init() {
	rootCmd.AddCommand(announceCmd)
}

var announceCmd = &cobra.Command{
	Use:   "announce",
	Short: "Export daily figures",
	Run: func(cmd *cobra.Command, args []string) {
		announce()
	},
}

func announce() {
	ctx := context.Background()

	data := generateImage(ctx)
	if data == nil {
		log.Debug("empty data")
		return
	}

	c, err := mastodon.SetupClient(ctx)
	if err != nil {
		panic(err)
	}
	imageID, err := c.UploadImage(ctx, imageFile)
	if err != nil {
		panic(err)
	}
	_, err = c.PostStatus(ctx, data.Status(), []string{imageID})
	if err != nil {
		panic(err)
	}
}

func generateImage(ctx context.Context) *homeassistant.RecorderDailyData {
	now := time.Now()
	c := homeassistant.CreateClient()
	c.InitWebsocket()
	dailyData := c.GetDailyData(now)
	if dailyData.IsEmpty() {
		return nil
	}
	fig := dailyData.ToFig()
	saveFigToJson(fig)
	figJsonToImage()
	return dailyData
}

func saveFigToJson(fig *grob.Fig) {
	figBytes, err := json.Marshal(fig)
	if err != nil {
		panic(err)
	}
	f, err := os.OpenFile(jsonFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		panic(err)
	}
	if _, err := f.Write(figBytes); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}

func figJsonToImage() {
	cmd := exec.Command("python", "plotly_image.py", jsonFile, imageFile)
	if err := cmd.Run(); err != nil {
		panic(err)
	}

}
