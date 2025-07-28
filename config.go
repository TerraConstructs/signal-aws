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
	InstanceID     string
	Retries        int
	PublishTimeout time.Duration
	Timeout        time.Duration
	LogFormat      string
	LogLevel       string
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
	flag.StringVar(&cfg.InstanceID, "instance-id", "", "override instance ID (default: fetch from IMDS)")
	flag.StringVar(&cfg.InstanceID, "n", "", "override instance ID (default: fetch from IMDS)")
	flag.IntVar(&cfg.Retries, "retries", 3, "transient-error retries")
	flag.DurationVar(&cfg.PublishTimeout, "publish-timeout", 10*time.Second, "timeout per SendMessage")
	flag.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "total operation timeout")
	flag.StringVar(&cfg.LogFormat, "log-format", "console", "log format: json or console")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "log level: debug, info, warn, or error")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `USAGE:
  tcsignal-aws [flags]

FLAGS:
  -u, --queue-url string     (required) SQS queue URL
  -i, --id string            (required) unique signal ID for the deployment
  -e, --exec string          run this command and signal based on its exit code
  -s, --status string        shortcut: send "SUCCESS" or "FAILURE" without exec
  -n, --instance-id string   override instance ID (default: fetch from IMDS)
  --retries int              transient-error retries (default 3)
  --publish-timeout duration timeout per SendMessage (default 10s)
  --timeout duration         total operation timeout (default 30s)
  --log-format string        log format: json or console (default "console")
  --log-level string         log level: debug, info, warn, or error (default "info")
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

	// Validate --log-format values
	if cfg.LogFormat != "json" && cfg.LogFormat != "console" {
		return nil, fmt.Errorf("--log-format must be either json or console")
	}

	// Validate --log-level values
	if cfg.LogLevel != "debug" && cfg.LogLevel != "info" && cfg.LogLevel != "warn" && cfg.LogLevel != "error" {
		return nil, fmt.Errorf("--log-level must be one of: debug, info, warn, error")
	}

	return &cfg, nil
}
