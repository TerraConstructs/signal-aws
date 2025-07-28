package signal

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type Config struct {
	QueueURL       string
	ID             string
	Exec           string
	Status         string
	Verbose        bool
	Retries        int
	PublishTimeout time.Duration
	Timeout        time.Duration
}

func ParseConfig() (*Config, error) {
	var cfg Config

	flag.StringVar(&cfg.QueueURL, "queue-url", "", "(required) SQS queue URL")
	flag.StringVar(&cfg.QueueURL, "u", "", "(required) SQS queue URL")
	flag.StringVar(&cfg.ID, "id", "", "(required) unique signal ID for the deployment")
	flag.StringVar(&cfg.ID, "i", "", "(required) unique signal ID for the deployment")
	flag.StringVar(&cfg.Exec, "exec", "", "run this command and signal based on its exit code")
	flag.StringVar(&cfg.Exec, "e", "", "run this command and signal based on its exit code")
	flag.StringVar(&cfg.Status, "status", "", "shortcut: send SUCCESS or FAILURE without exec")
	flag.StringVar(&cfg.Status, "s", "", "shortcut: send SUCCESS or FAILURE without exec")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "basic log verbosity")
	flag.BoolVar(&cfg.Verbose, "v", false, "basic log verbosity")
	flag.IntVar(&cfg.Retries, "retries", 3, "transient-error retries")
	flag.DurationVar(&cfg.PublishTimeout, "publish-timeout", 10*time.Second, "timeout per SendMessage")
	flag.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "total operation timeout")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `USAGE:
  tcons-signal [flags]

FLAGS:
  -u, --queue-url string     (required) SQS queue URL
  -i, --id string            (required) unique signal ID for the deployment
  -e, --exec string          run this command and signal based on its exit code
  -s, --status string        shortcut: send "SUCCESS" or "FAILURE" without exec
  -v, --verbose bool         basic log verbosity
  --retries int              transient-error retries (default 3)
  --publish-timeout duration timeout per SendMessage (default 10s)
  --timeout duration         total operation timeout (default 30s)
  --help                     show usage
`)
	}

	flag.Parse()

	// Validate required flags
	if cfg.QueueURL == "" {
		return nil, fmt.Errorf("--queue-url is required")
	}

	if cfg.ID == "" {
		return nil, fmt.Errorf("--id is required")
	}

	// Validate that either --exec or --status is provided
	if cfg.Exec == "" && cfg.Status == "" {
		return nil, fmt.Errorf("either --exec or --status must be provided")
	}

	// Validate --status values if provided
	if cfg.Status != "" && cfg.Status != "SUCCESS" && cfg.Status != "FAILURE" {
		return nil, fmt.Errorf("--status must be either SUCCESS or FAILURE")
	}

	return &cfg, nil
}
