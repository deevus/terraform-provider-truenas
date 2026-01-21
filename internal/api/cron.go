package api

// CronJobResponse represents a cron job from the TrueNAS API.
type CronJobResponse struct {
	ID          int64            `json:"id"`
	User        string           `json:"user"`
	Command     string           `json:"command"`
	Description string           `json:"description"`
	Enabled     bool             `json:"enabled"`
	Stdout      bool             `json:"stdout"`
	Stderr      bool             `json:"stderr"`
	Schedule    ScheduleResponse `json:"schedule"`
}
