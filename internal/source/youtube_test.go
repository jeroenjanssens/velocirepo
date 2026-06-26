package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestYouTubeFetchChannel(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("forHandle") != "" {
			json.NewEncoder(w).Encode(channelListResponse{
				Items: []channelItem{{ID: "UCsBjURrPoezykLs9EqgamOA"}},
			})
			return
		}
		json.NewEncoder(w).Encode(channelListResponse{
			Items: []channelItem{{
				ID: "UCsBjURrPoezykLs9EqgamOA",
				Statistics: channelStats{
					ViewCount:       1000000,
					SubscriberCount: 50000,
					VideoCount:      100,
				},
			}},
		})
	})

	mux.HandleFunc("/playlistItems", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(playlistItemsResponse{
			Items: []playlistItem{
				{ContentDetails: struct {
					VideoID string `json:"videoId"`
				}{VideoID: "vid1"}},
				{ContentDetails: struct {
					VideoID string `json:"videoId"`
				}{VideoID: "vid2"}},
			},
		})
	})

	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		likes1 := int64(10)
		comments1 := int64(5)
		likes2 := int64(20)
		json.NewEncoder(w).Encode(videoListResponse{
			Items: []videoItem{
				{
					ID:             "vid1",
					Snippet:        videoSnippet{Title: "Video One", PublishedAt: "2025-01-01T10:00:00Z", Tags: []string{"go", "tutorial"}},
					ContentDetails: videoContentDetails{Duration: "PT5M30S"},
					Statistics:     videoStats{ViewCount: 500, LikeCount: &likes1, CommentCount: &comments1},
				},
				{
					ID:             "vid2",
					Snippet:        videoSnippet{Title: "Video Two", PublishedAt: "2025-02-01T10:00:00Z"},
					ContentDetails: videoContentDetails{Duration: "PT10M0S"},
					Statistics:     videoStats{ViewCount: 300, LikeCount: &likes2, CommentCount: nil},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	yt := &YouTube{
		Client:  srv.Client(),
		APIKey:  "test-key",
		Target:  "@TestChannel",
		BaseURL: srv.URL,
	}

	records, err := yt.Fetch(context.Background(), FetchOptions{
		ProjectID: "test-proj",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Channel stats: subscribers, channel_views, video_count = 3
	// Video stats: vid1 (views, likes, comments) + vid2 (views, likes) = 5
	// vid2 has no comments (nil) so no comment record
	if len(records) != 8 {
		t.Fatalf("expected 8 records, got %d", len(records))
	}

	// Check channel-level records have no tags
	channelRecords := 0
	for _, r := range records {
		if len(r.Tags) == 0 {
			channelRecords++
		}
	}
	if channelRecords != 3 {
		t.Errorf("expected 3 channel-level records, got %d", channelRecords)
	}

	// Check target is always the config value
	for _, r := range records {
		if r.Target != "@TestChannel" {
			t.Errorf("expected target @TestChannel, got %s", r.Target)
		}
	}

	// Check content entries
	entries := yt.ContentEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 content entries, got %d", len(entries))
	}
	if entries[0].ID != "vid1" || entries[0].Title != "Video One" {
		t.Errorf("unexpected content entry 0: %+v", entries[0])
	}
	if entries[0].Duration == nil || *entries[0].Duration != 330 {
		t.Errorf("expected duration 330, got %v", entries[0].Duration)
	}
	if len(entries[0].Tags) != 2 || entries[0].Tags[0] != "go" {
		t.Errorf("expected tags [go, tutorial], got %v", entries[0].Tags)
	}
	if entries[1].Target != "@TestChannel" {
		t.Errorf("expected target @TestChannel, got %s", entries[1].Target)
	}
	if entries[1].Duration == nil || *entries[1].Duration != 600 {
		t.Errorf("expected duration 600, got %v", entries[1].Duration)
	}
	if entries[0].Source != "youtube" || entries[0].Type != "video" {
		t.Errorf("expected source=youtube type=video, got %+v", entries[0])
	}
	if entries[0].URL != "https://youtube.com/watch?v=vid1" {
		t.Errorf("expected URL for vid1, got %s", entries[0].URL)
	}
}

func TestYouTubeFetchSingleVideo(t *testing.T) {
	mux := http.NewServeMux()

	likes := int64(100)
	comments := int64(50)
	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(videoListResponse{
			Items: []videoItem{
				{
					ID:             "dQw4w9WgXcQ",
					Snippet:        videoSnippet{Title: "Never Gonna Give You Up", PublishedAt: "2009-10-25T06:57:33Z", Tags: []string{"rick astley", "music"}},
					ContentDetails: videoContentDetails{Duration: "PT3M33S"},
					Statistics:     videoStats{ViewCount: 1784285014, LikeCount: &likes, CommentCount: &comments},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	yt := &YouTube{
		Client:  srv.Client(),
		APIKey:  "test-key",
		Target:  "dQw4w9WgXcQ",
		BaseURL: srv.URL,
	}

	records, err := yt.Fetch(context.Background(), FetchOptions{
		ProjectID: "rickroll",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records (views, likes, comments), got %d", len(records))
	}

	if records[0].Metric != "total_views" || records[0].Value != 1784285014 {
		t.Errorf("unexpected views record: %+v", records[0])
	}
	if records[0].Tags["video_id"] != "dQw4w9WgXcQ" {
		t.Errorf("expected video_id tag, got %v", records[0].Tags)
	}

	entries := yt.ContentEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 content entry, got %d", len(entries))
	}
	if entries[0].Title != "Never Gonna Give You Up" {
		t.Errorf("unexpected title: %s", entries[0].Title)
	}
}

func TestYouTubeFetchPlaylist(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/playlistItems", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(playlistItemsResponse{
			Items: []playlistItem{
				{ContentDetails: struct {
					VideoID string `json:"videoId"`
				}{VideoID: "abc12345678"}},
			},
		})
	})

	likes := int64(5)
	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(videoListResponse{
			Items: []videoItem{
				{
					ID:             "abc12345678",
					Snippet:        videoSnippet{Title: "Playlist Video", PublishedAt: "2025-03-01T10:00:00Z"},
					ContentDetails: videoContentDetails{Duration: "PT2M0S"},
					Statistics:     videoStats{ViewCount: 999, LikeCount: &likes, CommentCount: nil},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	yt := &YouTube{
		Client:  srv.Client(),
		APIKey:  "test-key",
		Target:  "PLaBcDeFgHiJkLmNoPqRsTuVwXyZ012345",
		BaseURL: srv.URL,
	}

	records, err := yt.Fetch(context.Background(), FetchOptions{
		ProjectID: "my-playlist",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	// views + likes (no comments because nil)
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestYouTubePagination(t *testing.T) {
	mux := http.NewServeMux()
	callCount := 0

	mux.HandleFunc("/playlistItems", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Query().Get("pageToken") == "" {
			json.NewEncoder(w).Encode(playlistItemsResponse{
				Items: []playlistItem{
					{ContentDetails: struct {
						VideoID string `json:"videoId"`
					}{VideoID: "page1vid"}},
				},
				NextPageToken: "page2token",
			})
		} else {
			json.NewEncoder(w).Encode(playlistItemsResponse{
				Items: []playlistItem{
					{ContentDetails: struct {
						VideoID string `json:"videoId"`
					}{VideoID: "page2vid"}},
				},
			})
		}
	})

	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		likes := int64(1)
		json.NewEncoder(w).Encode(videoListResponse{
			Items: []videoItem{
				{ID: "page1vid", Snippet: videoSnippet{Title: "P1", PublishedAt: "2025-01-01T00:00:00Z"}, ContentDetails: videoContentDetails{Duration: "PT1M"}, Statistics: videoStats{ViewCount: 10, LikeCount: &likes}},
				{ID: "page2vid", Snippet: videoSnippet{Title: "P2", PublishedAt: "2025-01-02T00:00:00Z"}, ContentDetails: videoContentDetails{Duration: "PT2M"}, Statistics: videoStats{ViewCount: 20, LikeCount: &likes}},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	yt := &YouTube{
		Client:  srv.Client(),
		APIKey:  "test-key",
		Target:  "PLtestplaylist1234567890123456789012",
		BaseURL: srv.URL,
	}

	records, err := yt.Fetch(context.Background(), FetchOptions{
		ProjectID: "paged",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 playlist API calls (pagination), got %d", callCount)
	}

	// 2 videos * 2 metrics (views + likes) = 4
	if len(records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(records))
	}
}

func TestYouTubeNullableLikesComments(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(videoListResponse{
			Items: []videoItem{
				{
					ID:             "nolikescomm",
					Snippet:        videoSnippet{Title: "Hidden", PublishedAt: "2025-01-01T00:00:00Z"},
					ContentDetails: videoContentDetails{Duration: "PT4M"},
					Statistics:     videoStats{ViewCount: 100, LikeCount: nil, CommentCount: nil},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	yt := &YouTube{
		Client:  srv.Client(),
		APIKey:  "test-key",
		Target:  "nolikescomm",
		BaseURL: srv.URL,
	}

	records, err := yt.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Only views (likes and comments are nil)
	if len(records) != 1 {
		t.Fatalf("expected 1 record (views only), got %d", len(records))
	}
	if records[0].Metric != "total_views" {
		t.Errorf("expected total_views metric, got %s", records[0].Metric)
	}
}

func TestDetectYouTubeType(t *testing.T) {
	tests := []struct {
		target   string
		expected string
	}{
		{"@Fireship", "channel"},
		{"UCsBjURrPoezykLs9EqgamOA", "channel"},
		{"PLaBcDeFgHiJkLmNoPqRsTuVwXyZ012345", "playlist"},
		{"dQw4w9WgXcQ", "video"},
	}

	for _, tc := range tests {
		got := detectYouTubeType(tc.target)
		if got != tc.expected {
			t.Errorf("detectYouTubeType(%q) = %q, want %q", tc.target, got, tc.expected)
		}
	}
}
