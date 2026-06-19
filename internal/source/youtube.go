package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type YouTubeIndexEntry struct {
	VideoID     string   `json:"video_id"`
	Title       string   `json:"title"`
	PublishedAt string   `json:"published_at"`
	Channel     string   `json:"channel"`
	Duration    int64    `json:"duration"`
	Tags        []string `json:"tags,omitempty"`
}

type YouTube struct {
	Client  *http.Client
	APIKey  string
	Target  string
	BaseURL string

	indexEntries []YouTubeIndexEntry
}

func (y *YouTube) Name() string { return "youtube" }

func (y *YouTube) IndexEntries() []YouTubeIndexEntry {
	return y.indexEntries
}

func (y *YouTube) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	y.indexEntries = nil

	targetType := detectYouTubeType(y.Target)
	switch targetType {
	case "channel":
		return y.fetchChannel(ctx, opts)
	case "playlist":
		return y.fetchPlaylist(ctx, opts)
	case "video":
		return y.fetchVideos(ctx, opts, []string{y.Target})
	default:
		return nil, fmt.Errorf("cannot determine YouTube target type for %q", y.Target)
	}
}

func (y *YouTube) fetchChannel(ctx context.Context, opts FetchOptions) ([]Record, error) {
	channelID, err := y.resolveChannelID(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve channel: %w", err)
	}

	var records []Record

	channelRecords, err := y.fetchChannelStats(ctx, opts, channelID)
	if err != nil {
		return nil, fmt.Errorf("channel stats: %w", err)
	}
	records = append(records, channelRecords...)

	uploadsPlaylistID := "UU" + channelID[2:]
	videoIDs, err := y.fetchPlaylistVideoIDs(ctx, uploadsPlaylistID)
	if err != nil {
		return nil, fmt.Errorf("list videos: %w", err)
	}

	videoRecords, err := y.fetchVideos(ctx, opts, videoIDs)
	if err != nil {
		return nil, fmt.Errorf("video stats: %w", err)
	}
	records = append(records, videoRecords...)

	return records, nil
}

func (y *YouTube) fetchPlaylist(ctx context.Context, opts FetchOptions) ([]Record, error) {
	videoIDs, err := y.fetchPlaylistVideoIDs(ctx, y.Target)
	if err != nil {
		return nil, fmt.Errorf("list playlist videos: %w", err)
	}

	return y.fetchVideos(ctx, opts, videoIDs)
}

func (y *YouTube) resolveChannelID(ctx context.Context) (string, error) {
	target := y.Target
	if strings.HasPrefix(target, "UC") && len(target) == 24 {
		return target, nil
	}

	handle := target
	if !strings.HasPrefix(handle, "@") {
		handle = "@" + handle
	}

	url := fmt.Sprintf("%s/channels?part=id&forHandle=%s&key=%s", y.baseURL(), handle, y.APIKey)
	var resp channelListResponse
	if err := y.get(ctx, url, &resp); err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("channel not found for handle %q", y.Target)
	}
	return resp.Items[0].ID, nil
}

func (y *YouTube) fetchChannelStats(ctx context.Context, opts FetchOptions, channelID string) ([]Record, error) {
	url := fmt.Sprintf("%s/channels?part=statistics&id=%s&key=%s", y.baseURL(), channelID, y.APIKey)
	var resp channelListResponse
	if err := y.get(ctx, url, &resp); err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("channel %s not found", channelID)
	}

	stats := resp.Items[0].Statistics
	date := opts.EndDate.Format("2006-01-02")

	var records []Record

	if !stats.HiddenSubscriberCount {
		records = append(records, Record{
			Metric:    "total_subscribers",
			ProjectID: opts.ProjectID,
			Target:    y.Target,
			Date:      date,
			Value:     stats.SubscriberCount,
			Tags:      map[string]string{"video_id": ""},
		})
	}
	records = append(records, Record{
		Metric:    "total_channel_views",
		ProjectID: opts.ProjectID,
		Target:    y.Target,
		Date:      date,
		Value:     stats.ViewCount,
		Tags:      map[string]string{"video_id": ""},
	})
	records = append(records, Record{
		Metric:    "total_videos",
		ProjectID: opts.ProjectID,
		Target:    y.Target,
		Date:      date,
		Value:     stats.VideoCount,
		Tags:      map[string]string{"video_id": ""},
	})

	return records, nil
}

func (y *YouTube) fetchPlaylistVideoIDs(ctx context.Context, playlistID string) ([]string, error) {
	var videoIDs []string
	pageToken := ""

	for {
		url := fmt.Sprintf("%s/playlistItems?part=contentDetails&playlistId=%s&maxResults=50&key=%s",
			y.baseURL(), playlistID, y.APIKey)
		if pageToken != "" {
			url += "&pageToken=" + pageToken
		}

		var resp playlistItemsResponse
		if err := y.get(ctx, url, &resp); err != nil {
			return nil, err
		}

		for _, item := range resp.Items {
			videoIDs = append(videoIDs, item.ContentDetails.VideoID)
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return videoIDs, nil
}

func (y *YouTube) fetchVideos(ctx context.Context, opts FetchOptions, videoIDs []string) ([]Record, error) {
	var records []Record
	date := opts.EndDate.Format("2006-01-02")

	for i := 0; i < len(videoIDs); i += 50 {
		end := i + 50
		if end > len(videoIDs) {
			end = len(videoIDs)
		}
		batch := videoIDs[i:end]

		url := fmt.Sprintf("%s/videos?part=statistics,snippet,contentDetails&id=%s&key=%s",
			y.baseURL(), strings.Join(batch, ","), y.APIKey)

		var resp videoListResponse
		if err := y.get(ctx, url, &resp); err != nil {
			return nil, err
		}

		for _, item := range resp.Items {
			tags := map[string]string{"video_id": item.ID}

			records = append(records, Record{
				Metric:    "total_views",
				ProjectID: opts.ProjectID,
				Target:    y.Target,
				Date:      date,
				Value:     item.Statistics.ViewCount,
				Tags:      tags,
			})

			if item.Statistics.LikeCount != nil {
				records = append(records, Record{
					Metric:    "total_likes",
					ProjectID: opts.ProjectID,
					Target:    y.Target,
					Date:      date,
					Value:     *item.Statistics.LikeCount,
					Tags:      copyTags(tags),
				})
			}

			if item.Statistics.CommentCount != nil {
				records = append(records, Record{
					Metric:    "total_comments",
					ProjectID: opts.ProjectID,
					Target:    y.Target,
					Date:      date,
					Value:     *item.Statistics.CommentCount,
					Tags:      copyTags(tags),
				})
			}

			y.indexEntries = append(y.indexEntries, YouTubeIndexEntry{
				VideoID:     item.ID,
				Title:       item.Snippet.Title,
				PublishedAt: item.Snippet.PublishedAt,
				Channel:     y.Target,
				Duration:    parseISO8601Duration(item.ContentDetails.Duration),
				Tags:        item.Snippet.Tags,
			})
		}
	}

	return records, nil
}

func (y *YouTube) baseURL() string {
	if y.BaseURL != "" {
		return y.BaseURL
	}
	return "https://www.googleapis.com/youtube/v3"
}

func (y *YouTube) get(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := y.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("youtube API returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	return json.Unmarshal(body, result)
}

func detectYouTubeType(target string) string {
	if strings.HasPrefix(target, "@") {
		return "channel"
	}
	if strings.HasPrefix(target, "UC") && len(target) == 24 {
		return "channel"
	}
	if strings.HasPrefix(target, "PL") {
		return "playlist"
	}
	if len(target) == 11 {
		return "video"
	}
	if strings.HasPrefix(target, "@") || len(target) > 11 {
		return "channel"
	}
	return ""
}

func parseISO8601Duration(d string) int64 {
	var total int64
	d = strings.TrimPrefix(d, "PT")
	var num string
	for _, c := range d {
		switch c {
		case 'H':
			if n := parseInt(num); n > 0 {
				total += n * 3600
			}
			num = ""
		case 'M':
			if n := parseInt(num); n > 0 {
				total += n * 60
			}
			num = ""
		case 'S':
			if n := parseInt(num); n > 0 {
				total += n
			}
			num = ""
		default:
			num += string(c)
		}
	}
	return total
}

func parseInt(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}

func copyTags(tags map[string]string) map[string]string {
	m := make(map[string]string, len(tags))
	for k, v := range tags {
		m[k] = v
	}
	return m
}

// API response types

type channelListResponse struct {
	Items []channelItem `json:"items"`
}

type channelItem struct {
	ID             string          `json:"id"`
	Statistics     channelStats    `json:"statistics"`
	ContentDetails *channelContent `json:"contentDetails"`
}

type channelStats struct {
	ViewCount             int64 `json:"viewCount,string"`
	SubscriberCount       int64 `json:"subscriberCount,string"`
	VideoCount            int64 `json:"videoCount,string"`
	HiddenSubscriberCount bool  `json:"hiddenSubscriberCount"`
}

type channelContent struct {
	RelatedPlaylists struct {
		Uploads string `json:"uploads"`
	} `json:"relatedPlaylists"`
}

type playlistItemsResponse struct {
	Items         []playlistItem `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
}

type playlistItem struct {
	ContentDetails struct {
		VideoID string `json:"videoId"`
	} `json:"contentDetails"`
}

type videoListResponse struct {
	Items []videoItem `json:"items"`
}

type videoItem struct {
	ID             string              `json:"id"`
	Snippet        videoSnippet        `json:"snippet"`
	ContentDetails videoContentDetails `json:"contentDetails"`
	Statistics     videoStats          `json:"statistics"`
}

type videoSnippet struct {
	Title       string   `json:"title"`
	PublishedAt string   `json:"publishedAt"`
	Tags        []string `json:"tags"`
}

type videoContentDetails struct {
	Duration string `json:"duration"`
}

type videoStats struct {
	ViewCount    int64  `json:"viewCount,string"`
	LikeCount    *int64 `json:"likeCount,string"`
	CommentCount *int64 `json:"commentCount,string"`
}
