package main

import (
	"github.com/domgoodwin/go-automation/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	cmd.Execute()
}
