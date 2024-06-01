package main

import (
	"os"

	"github.com/domgoodwin/go-automation/cmd"
	log "github.com/sirupsen/logrus"
)

func main() {
	level, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
	host, _ := os.Hostname()
	log.SetFormatter(customLogger{
		host:      host,
		formatter: log.StandardLogger().Formatter,
	})
	log.Info("Starting up")
	cmd.Execute()
}

type customLogger struct {
	host      string
	formatter log.Formatter
}

func (l customLogger) Format(entry *log.Entry) ([]byte, error) {
	entry.Data["host"] = l.host
	return l.formatter.Format(entry)
}
