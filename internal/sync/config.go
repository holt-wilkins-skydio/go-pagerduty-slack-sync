package sync

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	SyncAllSchedKey        = "SYNC_ALL_SCHEDULES"
	scheduleKeyPrefix      = "SCHEDULE_"
	skipMissingGroupsKey   = "SKIP_MISSING_SLACK_GROUPS"
	noopKey                = "NOOP"
	pagerDutyTokenKey      = "PAGERDUTY_TOKEN"
	slackTokenKey          = "SLACK_TOKEN"
	runInterval            = "RUN_INTERVAL_SECONDS"
	pdScheduleLookaheadKey = "PAGERDUTY_SCHEDULE_LOOKAHEAD"
	runIntervalDefault     = 60
)

// Config is used to configure application
// PagerDutyToken - token used to connect to pagerduty API
// SlackToken - token used to connect to Slack API
type Config struct {
	SyncAllSchedules           bool
	SkipMissingGroups          bool
	Schedules                  []Schedule
	PagerDutyToken             string
	SlackToken                 string
	RunIntervalInSeconds       int
	PagerdutyScheduleLookahead time.Duration
	Noop                       bool
}

// Schedule models a PagerDuty schedule that will be synced with Slack
// ScheduleIDs - All PagerDuty schedule ID's to sync
// AllOnCallGroupName - Slack group name for all members of schedule
// CurrentOnCallGroupName - Slack group name for current person on call
type Schedule struct {
	ScheduleIDs            []string
	AllOnCallGroupName     string
	CurrentOnCallGroupName string
}

// NewStaticConfigFromEnv is a function to generate a only static config from env varibles
// PAGERDUTY_TOKEN - PagerDuty Token
// SLACK_TOKEN - Slack Token
func NewStaticConfigFromEnv() (*Config, error) {
	config := &Config{
		PagerDutyToken:       os.Getenv(pagerDutyTokenKey),
		SlackToken:           os.Getenv(slackTokenKey),
		SkipMissingGroups:    strings.ToLower(os.Getenv(skipMissingGroupsKey)) == "true",
		Noop:                 strings.ToLower(os.Getenv(noopKey)) == "true",
		RunIntervalInSeconds: runIntervalDefault,
	}

	runInterval := os.Getenv(runInterval)
	v, err := strconv.Atoi(runInterval)
	if err == nil {
		config.RunIntervalInSeconds = v
	}

	pagerdutyScheduleLookahead, err := getPagerdutyScheduleLookahead()
	if err != nil {
		return nil, err
	}
	config.PagerdutyScheduleLookahead = pagerdutyScheduleLookahead

	return config, nil
}

// NewConfigFromEnv is a function to generate a config from env varibles
// PAGERDUTY_TOKEN - PagerDuty Token
// SLACK_TOKEN - Slack Token
// SCHEDULE_XXX="id,name" e.g. 1234,platform-engineer will generate a schedule with the following values
// ScheduleID = "1234", AllOnCallGroupName = "all-oncall-platform-engineers", CurrentOnCallGroupName: "current-oncall-platform-engineer"
func NewConfigFromEnv() (*Config, error) {
	config, err := NewStaticConfigFromEnv()
	if err != nil {
		return nil, err
	}

	for _, key := range os.Environ() {
		if strings.HasPrefix(key, scheduleKeyPrefix) {
			value := strings.Split(key, "=")[1]
			scheduleValues := strings.Split(value, ",")
			if len(scheduleValues) != 2 {
				return nil, fmt.Errorf("expecting schedule value to be a comma separated scheduleId,name but got %s", value)
			}

			config.Schedules = appendSchedule(config.Schedules, scheduleValues[0], scheduleValues[1])
		}
	}

	if len(config.Schedules) == 0 {
		return nil, fmt.Errorf("expecting at least one schedule defined as an env var using prefix SCHEDULE_")
	}

	return config, nil
}

func appendSchedule(schedules []Schedule, scheduleID, teamName string) []Schedule {
	currentGroupName := fmt.Sprintf("current-oncall-%s", teamName)
	allGroupName := fmt.Sprintf("all-oncall-%ss", teamName)
	newScheduleList := make([]Schedule, len(schedules))
	updated := false

	for i, s := range schedules {
		if s.CurrentOnCallGroupName != currentGroupName {
			newScheduleList[i] = s

			continue
		}

		updated = true

		newScheduleList[i] = Schedule{
			ScheduleIDs:            append(s.ScheduleIDs, scheduleID),
			AllOnCallGroupName:     allGroupName,
			CurrentOnCallGroupName: currentGroupName,
		}
	}

	if !updated {
		newScheduleList = append(newScheduleList, Schedule{
			ScheduleIDs:            []string{scheduleID},
			AllOnCallGroupName:     allGroupName,
			CurrentOnCallGroupName: currentGroupName,
		})
	}

	return newScheduleList
}

func getPagerdutyScheduleLookahead() (time.Duration, error) {
	result := time.Hour * 24 * 100

	pdScheduleLookahead, ok := os.LookupEnv(pdScheduleLookaheadKey)
	if !ok {
		return result, nil
	}

	v, err := time.ParseDuration(pdScheduleLookahead)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s as time.Duration: %w", pdScheduleLookahead, err)
	}

	return v, nil
}

func (c *Config) populateSchedulesFromPagerduty(p *pagerDutyClient) error {
	pdSchedules, err := p.getAllSchedules()
	if err != nil {
		return err
	}
	for _, pds := range pdSchedules {
		c.Schedules = appendSchedule(c.Schedules, pds.ID, pds.Name)
	}
	return nil
}
