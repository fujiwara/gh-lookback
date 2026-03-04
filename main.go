package lookback

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"gopkg.in/yaml.v3"
)

// Run is the main entry point for the gh-lookback CLI.
func Run(ctx context.Context) error {
	var fromStr, toStr, host string

	now := time.Now()
	defaultFrom := now.AddDate(0, 0, -7).Format("2006-01-02")
	defaultTo := now.Format("2006-01-02")

	fs := flag.NewFlagSet("gh-lookback", flag.ContinueOnError)
	fs.StringVar(&fromStr, "from", defaultFrom, "Start date (YYYY-MM-DD)")
	fs.StringVar(&toStr, "to", defaultTo, "End date (YYYY-MM-DD)")
	fs.StringVar(&host, "host", "", "GitHub Enterprise host")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	fromDate, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return fmt.Errorf("invalid --from date %q: %w", fromStr, err)
	}
	toDate, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return fmt.Errorf("invalid --to date %q: %w", toStr, err)
	}

	clientOpts := api.ClientOptions{}
	if host != "" {
		clientOpts.Host = host
	}
	client, err := api.NewRESTClient(clientOpts)
	if err != nil {
		return fmt.Errorf("creating API client: %w", err)
	}

	user, err := GetCurrentUser(client)
	if err != nil {
		return err
	}

	if host == "" {
		host = "github.com"
	}

	opts := Options{
		From: fromDate,
		To:   toDate,
		Host: host,
	}

	result, err := Fetch(client, user, opts)
	if err != nil {
		return err
	}

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("encoding YAML: %w", err)
	}
	return enc.Close()
}
