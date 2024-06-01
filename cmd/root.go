package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

var cfgFile string
var userLicense string

func init() {
	viper.BindPFlag("author", rootCmd.PersistentFlags().Lookup("author"))
	viper.SetDefault("author", "Dom Goodwin git@dgood.win")
	viper.SetDefault("license", "apache")
}

var rootCmd = &cobra.Command{
	Use:   "gohome",
	Short: "Tool to automate tasks in my house",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("hello")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
