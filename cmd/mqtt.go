package cmd

import (
	"os"
	"os/signal"

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
	run := true
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Info("SIGINT received, stopping...")
			run = false
		}
	}()
	for run {
		log.Info("Subscribing...")
		err := mqtt.Subscribe(topic)
		if err != nil {
			log.Errorf("error during subscribe, retrying", err)
		}
	}
}
