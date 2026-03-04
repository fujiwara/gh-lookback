# gh-lookback

A GitHub CLI extension that retrieves your activity history (PRs created, PRs reviewed, Issues created) for a specified period and outputs it as YAML — useful for generating lookback summaries with LLMs.

## Installation

```bash
go install github.com/fujiwara/gh-lookback/cmd/gh-lookback@latest
```

## Usage

```bash
# Default: last 7 days
gh lookback

# Specify date range
gh lookback --from 2024-12-01 --to 2024-12-31

# GitHub Enterprise
gh lookback --host github.example.com --from 2024-12-01 --to 2024-12-31
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | 7 days ago | Start date (YYYY-MM-DD) |
| `--to` | today | End date (YYYY-MM-DD) |
| `--host` | (auto from gh config) | GitHub Enterprise host |

### Output

YAML output includes:

- **pull_requests_created** — PRs you authored
- **pull_requests_reviewed** — PRs you reviewed (excluding your own)
- **issues_created** — Issues you created

```yaml
lookback:
  user: fujiwara
  host: github.com
  from: "2024-01-01"
  to: "2024-01-07"
pull_requests_created:
  - title: "Add feature X"
    url: "https://github.com/org/repo/pull/123"
    repository: "org/repo"
    state: merged
    created_at: "2024-01-02T10:00:00Z"
    merged_at: "2024-01-03T15:00:00Z"
    labels:
      - enhancement
pull_requests_reviewed:
  - title: "Fix bug Y"
    url: "https://github.com/org/repo/pull/456"
    repository: "org/repo"
    state: open
    created_at: "2024-01-04T09:00:00Z"
issues_created:
  - title: "Bug report Z"
    url: "https://github.com/org/repo/issues/789"
    repository: "org/repo"
    state: open
    created_at: "2024-01-05T11:00:00Z"
    labels:
      - bug
```

## LICENSE

MIT

## Author

fujiwara
