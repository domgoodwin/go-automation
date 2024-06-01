package cmd

import (
	"github.com/domgoodwin/go-automation/mqtt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(subscribeCmd)
}

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to an MQTT topic",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		subscribe(args[0])
	},
}

func subscribe(topic string) {
	for {
		log.Info("Subscribing...")
		err := mqtt.Subscribe(topic)
		if err != nil {
			log.Errorf("error during subscribe, retrying", err)
		}
	}
}
