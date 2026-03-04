package lookback

import (
	"testing"
	"time"
)

func TestParseSearchResponse(t *testing.T) {
	data := []byte(`{
		"total_count": 2,
		"items": [
			{
				"title": "Add feature X",
				"html_url": "https://github.com/org/repo/pull/123",
				"state": "closed",
				"created_at": "2024-01-02T10:00:00Z",
				"repository_url": "https://api.github.com/repos/org/repo",
				"pull_request": {
					"merged_at": "2024-01-03T15:00:00Z"
				},
				"labels": [
					{"name": "enhancement"}
				]
			},
			{
				"title": "Fix bug Y",
				"html_url": "https://github.com/org/repo/pull/456",
				"state": "open",
				"created_at": "2024-01-04T09:00:00Z",
				"repository_url": "https://api.github.com/repos/org/repo",
				"labels": []
			}
		]
	}`)

	result, err := ParseSearchResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", result.TotalCount)
	}
	if len(result.Items) != 2 {
		t.Fatalf("Items length = %d, want 2", len(result.Items))
	}

	// First item
	item0 := result.Items[0]
	if item0.Title != "Add feature X" {
		t.Errorf("Items[0].Title = %q, want %q", item0.Title, "Add feature X")
	}
	if item0.HTMLURL != "https://github.com/org/repo/pull/123" {
		t.Errorf("Items[0].HTMLURL = %q", item0.HTMLURL)
	}
	if item0.PullRequestDetail == nil {
		t.Fatal("Items[0].PullRequestDetail is nil")
	}
	if item0.PullRequestDetail.MergedAt == nil {
		t.Fatal("Items[0].PullRequestDetail.MergedAt is nil")
	}
	if *item0.PullRequestDetail.MergedAt != "2024-01-03T15:00:00Z" {
		t.Errorf("MergedAt = %q", *item0.PullRequestDetail.MergedAt)
	}
	if len(item0.Labels) != 1 || item0.Labels[0].Name != "enhancement" {
		t.Errorf("Items[0].Labels = %v", item0.Labels)
	}

	// Second item (no pull_request detail)
	item1 := result.Items[1]
	if item1.PullRequestDetail != nil {
		t.Errorf("Items[1].PullRequestDetail should be nil, got %v", item1.PullRequestDetail)
	}
}

func TestExtractRepoFullName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://api.github.com/repos/org/repo", "org/repo"},
		{"https://api.github.com/repos/my-org/my-repo", "my-org/my-repo"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		got := extractRepoFullName(tt.input)
		if got != tt.want {
			t.Errorf("extractRepoFullName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveState(t *testing.T) {
	mergedAt := "2024-01-03T15:00:00Z"

	tests := []struct {
		name string
		item searchItem
		want string
	}{
		{
			name: "merged PR",
			item: searchItem{
				State: "closed",
				PullRequestDetail: &pullRequestDetail{
					MergedAt: &mergedAt,
				},
			},
			want: "merged",
		},
		{
			name: "closed PR (not merged)",
			item: searchItem{
				State: "closed",
				PullRequestDetail: &pullRequestDetail{
					MergedAt: nil,
				},
			},
			want: "closed",
		},
		{
			name: "open PR",
			item: searchItem{
				State: "open",
			},
			want: "open",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveState(tt.item)
			if got != tt.want {
				t.Errorf("resolveState() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultDates(t *testing.T) {
	now := time.Now()
	from := now.AddDate(0, 0, -7)

	fromStr := from.Format("2006-01-02")
	toStr := now.Format("2006-01-02")

	// Verify the dates are valid
	parsedFrom, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		t.Fatalf("failed to parse from date: %v", err)
	}
	parsedTo, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		t.Fatalf("failed to parse to date: %v", err)
	}

	diff := parsedTo.Sub(parsedFrom)
	if diff != 7*24*time.Hour {
		t.Errorf("date difference = %v, want 7 days", diff)
	}
}
