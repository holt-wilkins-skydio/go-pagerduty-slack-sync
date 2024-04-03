package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kevholditch/go-pagerduty-slack-sync/internal/sync"
	"github.com/sirupsen/logrus"
)

func main() {

	// Make the stop channel buffered
	stop := make(chan os.Signal, 1) // Buffer size of 1
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var config *sync.Config
	var err error

	if strings.ToLower(os.Getenv("SYNC_ALL_SCHEDULES")) == "true" {
		if config, err = sync.NewStaticConfigFromEnv(); err != nil {
			logrus.Errorf("could not parse config, error: %v", err)
			os.Exit(-1)
			return
		}
		config.SyncAllSchedules = true

		logrus.Infof("starting, going to sync all schedules")
	} else {
		if config, err = sync.NewConfigFromEnv(); err != nil {
			logrus.Errorf("could not parse config, error: %v", err)
			os.Exit(-1)
			return
		}

		logrus.Infof("starting, going to sync %d schedules", len(config.Schedules))
	}

	timer := time.NewTicker(time.Second * time.Duration(config.RunIntervalInSeconds))

	for alive := true; alive; {
		select {
		case <-stop:
			logrus.Infof("stopping...")
			alive = false
			os.Exit(0)
		case <-timer.C:
			err = sync.Schedules(config)
			if err != nil {
				logrus.Errorf("could not sync schedules, error: %v", err)
				os.Exit(-1)
				return
			}
		}
	}
}
