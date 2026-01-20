package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// JobState represents the state of a TrueNAS job.
type JobState string

const (
	JobStateRunning JobState = "RUNNING"
	JobStateSuccess JobState = "SUCCESS"
	JobStateFailed  JobState = "FAILED"
	JobStateWaiting JobState = "WAITING"
)

// Job represents a TrueNAS job from the middleware.
type Job struct {
	ID          int64           `json:"id"`
	State       JobState        `json:"state"`
	Error       string          `json:"error,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
	LogsExcerpt string          `json:"logs_excerpt,omitempty"`
	LogsPath    string          `json:"logs_path,omitempty"`
}

// JobPollerConfig configures the polling behavior.
type JobPollerConfig struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// DefaultJobPollerConfig returns sensible defaults for job polling.
func DefaultJobPollerConfig() *JobPollerConfig {
	return &JobPollerConfig{
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     10 * time.Second,
		Multiplier:      1.5,
	}
}

// JobPoller polls for job completion with exponential backoff.
type JobPoller struct {
	client Client
	config *JobPollerConfig
}

// NewJobPoller creates a new JobPoller with the given client and config.
// If config is nil, DefaultJobPollerConfig() is used.
func NewJobPoller(client Client, config *JobPollerConfig) *JobPoller {
	if config == nil {
		config = DefaultJobPollerConfig()
	}
	return &JobPoller{
		client: client,
		config: config,
	}
}

// Wait polls for job completion until success, failure, timeout, or context cancellation.
// Returns the job result on success, or an error on failure/timeout.
func (p *JobPoller) Wait(ctx context.Context, jobID int64, timeout time.Duration) (json.RawMessage, error) {
	deadline := time.Now().Add(timeout)
	interval := p.config.InitialInterval

	for {
		// Check timeout
		if time.Now().After(deadline) {
			return nil, NewTimeoutError(jobID, timeout.String())
		}

		// Check context
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Poll for job status
		job, err := p.getJob(ctx, jobID)
		if err != nil {
			return nil, err
		}

		switch job.State {
		case JobStateSuccess:
			return job.Result, nil
		case JobStateFailed:
			err := ParseTrueNASError(job.Error)
			err.LogsExcerpt = job.LogsExcerpt

			// Fetch app lifecycle log if applicable
			EnrichAppLifecycleError(ctx, err, func(ctx context.Context, path string) (string, error) {
				data, err := p.client.ReadFile(ctx, path)
				return string(data), err
			})

			return nil, err
		case JobStateRunning, JobStateWaiting:
			// Continue polling
		default:
			// Unknown state, continue polling
		}

		// Wait before next poll with exponential backoff
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		// Increase interval for next poll
		interval = time.Duration(float64(interval) * p.config.Multiplier)
		if interval > p.config.MaxInterval {
			interval = p.config.MaxInterval
		}
	}
}

// getJob fetches a job by ID from the TrueNAS API.
func (p *JobPoller) getJob(ctx context.Context, jobID int64) (*Job, error) {
	// Build filter: [["id", "=", jobID]]
	filter := []any{
		[]any{"id", "=", jobID},
	}

	result, err := p.client.Call(ctx, "core.get_jobs", []any{filter})
	if err != nil {
		return nil, err
	}

	var jobs []Job
	if err := json.Unmarshal(result, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse job response: %w", err)
	}

	if len(jobs) == 0 {
		return nil, &TrueNASError{
			Code:       "ENOENT",
			Message:    fmt.Sprintf("Job %d not found", jobID),
			JobID:      jobID,
			Suggestion: "The job may have expired or the ID is incorrect.",
		}
	}

	return &jobs[0], nil
}

// ParseJobID parses a job ID from a JSON response.
// Many TrueNAS operations return just the job ID as a number.
func ParseJobID(data json.RawMessage) (int64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("empty job ID data")
	}

	// Check for null explicitly
	if string(data) == "null" {
		return 0, fmt.Errorf("job ID is null")
	}

	var jobID int64
	if err := json.Unmarshal(data, &jobID); err != nil {
		return 0, fmt.Errorf("failed to parse job ID: %w", err)
	}

	return jobID, nil
}

