package lookback

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Options holds the CLI options for lookback.
type Options struct {
	From time.Time
	To   time.Time
	Host string
}

// LookbackResult is the top-level output structure.
type LookbackResult struct {
	Lookback             LookbackMeta  `yaml:"lookback"`
	PullRequestsCreated  []PullRequest `yaml:"pull_requests_created"`
	PullRequestsReviewed []PullRequest `yaml:"pull_requests_reviewed"`
	IssuesCreated        []Issue       `yaml:"issues_created"`
}

// LookbackMeta holds metadata about the lookback query.
type LookbackMeta struct {
	User string `yaml:"user"`
	Host string `yaml:"host"`
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// PullRequest represents a pull request item.
type PullRequest struct {
	Title      string   `yaml:"title"`
	URL        string   `yaml:"url"`
	Repository string   `yaml:"repository"`
	State      string   `yaml:"state"`
	CreatedAt  string   `yaml:"created_at"`
	MergedAt   string   `yaml:"merged_at,omitempty"`
	Labels     []string `yaml:"labels,omitempty"`
}

// Issue represents an issue item.
type Issue struct {
	Title      string   `yaml:"title"`
	URL        string   `yaml:"url"`
	Repository string   `yaml:"repository"`
	State      string   `yaml:"state"`
	CreatedAt  string   `yaml:"created_at"`
	Labels     []string `yaml:"labels,omitempty"`
}

// searchResult represents the GitHub Search API response.
type searchResult struct {
	TotalCount int          `json:"total_count"`
	Items      []searchItem `json:"items"`
}

type searchItem struct {
	Title             string             `json:"title"`
	HTMLURL           string             `json:"html_url"`
	State             string             `json:"state"`
	CreatedAt         string             `json:"created_at"`
	PullRequestDetail *pullRequestDetail `json:"pull_request,omitempty"`
	RepositoryURL     string             `json:"repository_url"`
	Labels            []labelItem        `json:"labels"`
}

type pullRequestDetail struct {
	MergedAt *string `json:"merged_at"`
}

type labelItem struct {
	Name string `json:"name"`
}

// extractRepoFullName extracts "owner/repo" from a repository_url like
// "https://api.github.com/repos/owner/repo".
func extractRepoFullName(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err != nil {
		return repoURL
	}
	// path is "/repos/owner/repo"
	parts := splitPath(u.Path)
	if len(parts) >= 3 && parts[0] == "repos" {
		return parts[1] + "/" + parts[2]
	}
	return repoURL
}

// splitPath splits a URL path into non-empty segments.
func splitPath(path string) []string {
	var parts []string
	for _, p := range split(path, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func split(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// resolveState determines the display state for a PR.
func resolveState(item searchItem) string {
	if item.PullRequestDetail != nil && item.PullRequestDetail.MergedAt != nil {
		return "merged"
	}
	return item.State
}

// GetCurrentUser fetches the authenticated user's login name.
func GetCurrentUser(client *api.RESTClient) (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := client.Get("user", &user); err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return user.Login, nil
}

// Fetch retrieves all activity data from GitHub Search API.
func Fetch(client *api.RESTClient, user string, opts Options) (*LookbackResult, error) {
	fromStr := opts.From.Format("2006-01-02")
	toStr := opts.To.Format("2006-01-02")
	dateRange := fromStr + ".." + toStr

	result := &LookbackResult{
		Lookback: LookbackMeta{
			User: user,
			Host: opts.Host,
			From: fromStr,
			To:   toStr,
		},
	}

	// PRs created by user
	slog.Info("Fetching PRs created ...", "user", user)
	prsCreated, err := searchAll(client, fmt.Sprintf("type:pr author:%s created:%s", user, dateRange))
	if err != nil {
		return nil, fmt.Errorf("searching PRs created: %w", err)
	}
	slog.Info("Found PRs created", "count", len(prsCreated))
	for _, item := range prsCreated {
		pr := PullRequest{
			Title:      item.Title,
			URL:        item.HTMLURL,
			Repository: extractRepoFullName(item.RepositoryURL),
			State:      resolveState(item),
			CreatedAt:  item.CreatedAt,
		}
		if item.PullRequestDetail != nil && item.PullRequestDetail.MergedAt != nil {
			pr.MergedAt = *item.PullRequestDetail.MergedAt
		}
		for _, l := range item.Labels {
			pr.Labels = append(pr.Labels, l.Name)
		}
		result.PullRequestsCreated = append(result.PullRequestsCreated, pr)
	}

	// PRs reviewed by user (excluding own PRs)
	slog.Info("Fetching PRs reviewed ...", "user", user)
	prsReviewed, err := searchAll(client, fmt.Sprintf("type:pr reviewed-by:%s -author:%s created:%s", user, user, dateRange))
	if err != nil {
		return nil, fmt.Errorf("searching PRs reviewed: %w", err)
	}
	slog.Info("Found PRs reviewed", "count", len(prsReviewed))
	for _, item := range prsReviewed {
		pr := PullRequest{
			Title:      item.Title,
			URL:        item.HTMLURL,
			Repository: extractRepoFullName(item.RepositoryURL),
			State:      resolveState(item),
			CreatedAt:  item.CreatedAt,
		}
		if item.PullRequestDetail != nil && item.PullRequestDetail.MergedAt != nil {
			pr.MergedAt = *item.PullRequestDetail.MergedAt
		}
		for _, l := range item.Labels {
			pr.Labels = append(pr.Labels, l.Name)
		}
		result.PullRequestsReviewed = append(result.PullRequestsReviewed, pr)
	}

	// Issues created by user
	slog.Info("Fetching issues created ...", "user", user)
	issuesCreated, err := searchAll(client, fmt.Sprintf("type:issue author:%s created:%s", user, dateRange))
	if err != nil {
		return nil, fmt.Errorf("searching issues created: %w", err)
	}
	slog.Info("Found issues created", "count", len(issuesCreated))
	for _, item := range issuesCreated {
		issue := Issue{
			Title:      item.Title,
			URL:        item.HTMLURL,
			Repository: extractRepoFullName(item.RepositoryURL),
			State:      item.State,
			CreatedAt:  item.CreatedAt,
		}
		for _, l := range item.Labels {
			issue.Labels = append(issue.Labels, l.Name)
		}
		result.IssuesCreated = append(result.IssuesCreated, issue)
	}

	return result, nil
}

// searchAll performs paginated search and returns all items.
func searchAll(client *api.RESTClient, query string) ([]searchItem, error) {
	var allItems []searchItem
	page := 1
	perPage := 100

	for {
		path := fmt.Sprintf("search/issues?q=%s&per_page=%d&page=%d",
			url.QueryEscape(query), perPage, page)

		resp, err := doSearch(client, path)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, resp.Items...)

		if len(allItems) >= resp.TotalCount || len(resp.Items) < perPage {
			break
		}
		slog.Info("Fetching more results ...", "fetched", len(allItems), "total", resp.TotalCount)
		page++
	}

	return allItems, nil
}

// doSearch performs a single search API request.
func doSearch(client *api.RESTClient, path string) (*searchResult, error) {
	resp, err := client.Request(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("search API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result searchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	return &result, nil
}

// ParseSearchResponse parses a JSON search response body into searchResult.
// Exported for testing.
func ParseSearchResponse(data []byte) (*searchResult, error) {
	var result searchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
