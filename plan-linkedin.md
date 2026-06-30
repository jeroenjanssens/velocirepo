# Plan: LinkedIn Fetcher

## Goal

Add a LinkedIn source that fetches post listings and engagement metrics from the LinkedIn Community Management API. Like YouTube, LinkedIn is a dual-output source: it produces both time-series metrics (engagement stats) and content entries (post listings).

## LinkedIn API Overview

LinkedIn uses the **Community Management API** (v2) for programmatic access to posts and their statistics.

### Authentication

- **OAuth 2.0** with a 3-legged flow
- Access tokens are **short-lived** (~60 days for member tokens, 365 days for 2-legged client credential grants on organization APIs)
- Requires a **LinkedIn Developer App** with the "Community Management" product approved
- Environment variable: `LINKEDIN_TOKEN`
- No API key — it's always a Bearer token

### Required Scopes

| Use case | Scope |
|----------|-------|
| Personal posts | `r_member_social` |
| Organization posts | `r_organization_social` |
| Post statistics | `r_organization_social` (orgs) or `r_member_social` (personal) |

### Relevant Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /rest/posts?author={urn}&count=100` | List posts by author |
| `GET /rest/organizationalEntityShareStatistics?organizationalEntity={urn}&shares=List({post_urns})` | Post-level engagement stats |
| `GET /rest/networkSizes/{urn}?edgeType=COMPANY_FOLLOWED_BY_MEMBER` | Follower count (orgs) |

### Rate Limits

- **100 requests/day** per application for most endpoints (Community Management)
- Pagination: max 100 posts per page with `start` and `count` params
- Response format: JSON with `elements` array

## Config

```toml
[projects.my-project]
linkedin = "urn:li:organization:12345678"
```

The target is a LinkedIn URN. Supported formats:
- `urn:li:organization:12345678` — company/organization page
- `urn:li:person:Ab1CdEfGhI` — personal profile (requires member token)

Shorthand alternative (resolved at fetch time):
- `company:my-company-slug` — resolved via vanity name lookup

## Source Struct

```go
// internal/source/linkedin.go

type LinkedIn struct {
    Client  *http.Client
    Token   string
    Target  string // URN or shorthand
    BaseURL string // for testing

    contentEntries []ContentEntry
}

func (l *LinkedIn) Name() string { return "linkedin" }

func (l *LinkedIn) ContentEntries() []ContentEntry {
    return l.contentEntries
}

func (l *LinkedIn) ContentFilename() string {
    return "posts.jsonl"
}
```

LinkedIn implements both `Source` (metrics) and `ContentProvider` (post listings).

## API Response Shapes

### List Posts Response

```json
{
  "elements": [
    {
      "id": "urn:li:share:7123456789012345678",
      "author": "urn:li:organization:12345678",
      "createdAt": 1719792000000,
      "lastModifiedAt": 1719792000000,
      "visibility": "PUBLIC",
      "commentary": "Excited to announce our new release! 🚀\n\nCheck it out at...",
      "content": {
        "article": {
          "source": "https://example.com/blog/new-release",
          "title": "New Release v2.0",
          "description": "We're excited to announce..."
        }
      },
      "lifecycleState": "PUBLISHED",
      "distribution": {
        "feedDistribution": "MAIN_FEED"
      }
    }
  ],
  "paging": {
    "count": 100,
    "start": 0,
    "total": 47
  }
}
```

### Share Statistics Response

```json
{
  "elements": [
    {
      "totalShareStatistics": {
        "shareCount": 12,
        "likeCount": 85,
        "commentCount": 14,
        "impressionCount": 4521,
        "clickCount": 234,
        "engagement": 0.0412,
        "uniqueImpressionsCount": 3890
      },
      "share": "urn:li:share:7123456789012345678",
      "organizationalEntity": "urn:li:organization:12345678"
    }
  ]
}
```

### Follower Count Response

```json
{
  "firstDegreeSize": 15420
}
```

## Data Output

### Content: `data/content/linkedin/<project>/posts.jsonl`

```json
{"source":"linkedin","target":"urn:li:organization:12345678","id":"urn:li:share:7123456789012345678","title":"Excited to announce our new release! 🚀","description":"Excited to announce our new release! 🚀\n\nCheck it out at...","published_at":"2025-06-30T12:00:00Z","url":"https://www.linkedin.com/feed/update/urn:li:share:7123456789012345678","tags":["release","announcement"],"type":"post"}
{"source":"linkedin","target":"urn:li:organization:12345678","id":"urn:li:share:7198765432109876543","title":"We're hiring! Looking for senior engineers...","description":"We're hiring! Looking for senior engineers to join our team...","published_at":"2025-06-25T09:30:00Z","url":"https://www.linkedin.com/feed/update/urn:li:share:7198765432109876543","tags":["hiring"],"type":"post","metadata":{"has_image":true}}
```

Field mapping from API:
- `id` ← `elements[].id` (the share/ugcPost URN)
- `title` ← first 100 chars of `commentary` (truncated at word boundary)
- `description` ← full `commentary` text
- `published_at` ← `createdAt` (epoch ms → ISO 8601)
- `url` ← `https://www.linkedin.com/feed/update/{id}`
- `tags` ← extracted `#hashtags` from commentary
- `type` ← derived from `content` field: `"post"` (text only), `"article"` (link/article), `"image"`, `"video"`, `"document"`
- `metadata` ← `{"has_image": true, "article_url": "..."}` (optional extras)

### Metrics: `data/metrics/linkedin/<project>/<date>.jsonl`

```json
{"source":"linkedin","metric":"total_impressions","project_id":"my-project","target":"urn:li:organization:12345678","date":"2025-06-30","value":4521,"tags":{"post_id":"urn:li:share:7123456789012345678"}}
{"source":"linkedin","metric":"total_likes","project_id":"my-project","target":"urn:li:organization:12345678","date":"2025-06-30","value":85,"tags":{"post_id":"urn:li:share:7123456789012345678"}}
{"source":"linkedin","metric":"total_comments","project_id":"my-project","target":"urn:li:organization:12345678","date":"2025-06-30","value":14,"tags":{"post_id":"urn:li:share:7123456789012345678"}}
{"source":"linkedin","metric":"total_shares","project_id":"my-project","target":"urn:li:organization:12345678","date":"2025-06-30","value":12,"tags":{"post_id":"urn:li:share:7123456789012345678"}}
{"source":"linkedin","metric":"total_clicks","project_id":"my-project","target":"urn:li:organization:12345678","date":"2025-06-30","value":234,"tags":{"post_id":"urn:li:share:7123456789012345678"}}
{"source":"linkedin","metric":"total_followers","project_id":"my-project","target":"urn:li:organization:12345678","date":"2025-06-30","value":15420}
```

Per-post metrics are tagged with `post_id` (same pattern as YouTube's `video_id` tag). The `total_followers` metric has no tag — it's an org-level aggregate.

## Fetch Logic

```go
func (l *LinkedIn) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
    l.contentEntries = nil

    // 1. Fetch all posts (paginated)
    posts, err := l.fetchPosts(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("fetch posts: %w", err)
    }

    // 2. Build content entries from posts
    for _, p := range posts {
        l.contentEntries = append(l.contentEntries, postToContentEntry(p, l.Target))
    }

    // 3. Fetch engagement stats for all posts (batched)
    var records []Record
    statsRecords, err := l.fetchPostStats(ctx, opts, posts)
    if err != nil {
        return nil, fmt.Errorf("fetch stats: %w", err)
    }
    records = append(records, statsRecords...)

    // 4. Fetch follower count (org only)
    if strings.Contains(l.Target, "organization") {
        followerRecords, err := l.fetchFollowerCount(ctx, opts)
        if err != nil {
            // Non-fatal: log and continue
            slog.Warn("linkedin follower count failed", "error", err)
        } else {
            records = append(records, followerRecords...)
        }
    }

    return records, nil
}
```

### Pagination

```go
func (l *LinkedIn) fetchPosts(ctx context.Context, opts FetchOptions) ([]linkedinPost, error) {
    var allPosts []linkedinPost
    start := 0

    for {
        url := fmt.Sprintf("%s/rest/posts?author=%s&count=100&start=%d&sortBy=LAST_MODIFIED",
            l.baseURL(), url.QueryEscape(l.Target), start)

        var resp postsResponse
        if err := l.get(ctx, url, &resp); err != nil {
            return nil, err
        }

        allPosts = append(allPosts, resp.Elements...)

        if len(resp.Elements) < resp.Paging.Count || start+len(resp.Elements) >= resp.Paging.Total {
            break
        }
        start += len(resp.Elements)
    }

    return allPosts, nil
}
```

### Batched Stats

The statistics endpoint accepts up to 20 share URNs at once:

```go
func (l *LinkedIn) fetchPostStats(ctx context.Context, opts FetchOptions, posts []linkedinPost) ([]Record, error) {
    var records []Record
    date := opts.EndDate.Format("2006-01-02")

    // Batch in groups of 20
    for i := 0; i < len(posts); i += 20 {
        end := i + 20
        if end > len(posts) {
            end = len(posts)
        }
        batch := posts[i:end]

        // Build shares query param: shares=List(urn1,urn2,...)
        var urns []string
        for _, p := range batch {
            urns = append(urns, p.ID)
        }
        sharesParam := "List(" + strings.Join(urns, ",") + ")"

        url := fmt.Sprintf("%s/rest/organizationalEntityShareStatistics?organizationalEntity=%s&shares=%s",
            l.baseURL(), url.QueryEscape(l.Target), url.QueryEscape(sharesParam))

        var resp statsResponse
        if err := l.get(ctx, url, &resp); err != nil {
            return nil, err
        }

        for _, stat := range resp.Elements {
            tags := map[string]string{"post_id": stat.Share}
            records = append(records,
                Record{Metric: "total_impressions", ProjectID: opts.ProjectID, Target: l.Target, Date: date, Value: stat.Stats.ImpressionCount, Tags: tags},
                Record{Metric: "total_likes", ProjectID: opts.ProjectID, Target: l.Target, Date: date, Value: stat.Stats.LikeCount, Tags: copyTags(tags)},
                Record{Metric: "total_comments", ProjectID: opts.ProjectID, Target: l.Target, Date: date, Value: stat.Stats.CommentCount, Tags: copyTags(tags)},
                Record{Metric: "total_shares", ProjectID: opts.ProjectID, Target: l.Target, Date: date, Value: stat.Stats.ShareCount, Tags: copyTags(tags)},
                Record{Metric: "total_clicks", ProjectID: opts.ProjectID, Target: l.Target, Date: date, Value: stat.Stats.ClickCount, Tags: copyTags(tags)},
            )
        }
    }

    return records, nil
}
```

## HTTP Helper

LinkedIn API requires specific headers:

```go
func (l *LinkedIn) get(ctx context.Context, url string, result interface{}) error {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", "Bearer "+l.Token)
    req.Header.Set("LinkedIn-Version", "202406")
    req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

    resp, err := l.Client.Do(req)
    if err != nil {
        return fmt.Errorf("request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("linkedin API returned %d: %s", resp.StatusCode, string(body))
    }

    return json.NewDecoder(resp.Body).Decode(result)
}

func (l *LinkedIn) baseURL() string {
    if l.BaseURL != "" {
        return l.BaseURL
    }
    return "https://api.linkedin.com"
}
```

## API Response Types

```go
type linkedinPost struct {
    ID           string              `json:"id"`
    Author       string              `json:"author"`
    CreatedAt    int64               `json:"createdAt"`
    Commentary   string              `json:"commentary"`
    Content      *linkedinContent    `json:"content"`
    Visibility   string              `json:"visibility"`
    Lifecycle    string              `json:"lifecycleState"`
}

type linkedinContent struct {
    Article  *linkedinArticle `json:"article"`
    Media    *linkedinMedia   `json:"media"`
}

type linkedinArticle struct {
    Source      string `json:"source"`
    Title       string `json:"title"`
    Description string `json:"description"`
}

type linkedinMedia struct {
    ID string `json:"id"`
}

type postsResponse struct {
    Elements []linkedinPost `json:"elements"`
    Paging   struct {
        Count int `json:"count"`
        Start int `json:"start"`
        Total int `json:"total"`
    } `json:"paging"`
}

type shareStats struct {
    ImpressionCount int64   `json:"impressionCount"`
    LikeCount       int64   `json:"likeCount"`
    CommentCount    int64   `json:"commentCount"`
    ShareCount      int64   `json:"shareCount"`
    ClickCount      int64   `json:"clickCount"`
    Engagement      float64 `json:"engagement"`
}

type statsElement struct {
    Stats shareStats `json:"totalShareStatistics"`
    Share string     `json:"share"`
}

type statsResponse struct {
    Elements []statsElement `json:"elements"`
}

type followerResponse struct {
    FirstDegreeSize int64 `json:"firstDegreeSize"`
}
```

## Content Entry Conversion

```go
func postToContentEntry(p linkedinPost, target string) ContentEntry {
    publishedAt := time.UnixMilli(p.CreatedAt).UTC().Format(time.RFC3339)

    // Title: first 100 chars of commentary at word boundary
    title := p.Commentary
    if len(title) > 100 {
        title = title[:100]
        if idx := strings.LastIndex(title, " "); idx > 50 {
            title = title[:idx]
        }
        title += "..."
    }

    // Extract hashtags
    var tags []string
    for _, match := range hashtagRe.FindAllString(p.Commentary, -1) {
        tags = append(tags, strings.TrimPrefix(match, "#"))
    }

    // Determine post type
    postType := "post"
    var metadata map[string]any
    if p.Content != nil {
        if p.Content.Article != nil {
            postType = "article"
            metadata = map[string]any{"article_url": p.Content.Article.Source}
        } else if p.Content.Media != nil {
            postType = "media"
        }
    }

    return ContentEntry{
        Source:      "linkedin",
        Target:      target,
        ID:          p.ID,
        Title:       title,
        Description: p.Commentary,
        PublishedAt: publishedAt,
        URL:         "https://www.linkedin.com/feed/update/" + p.ID,
        Tags:        tags,
        Type:        postType,
        Metadata:    metadata,
    }
}

var hashtagRe = regexp.MustCompile(`#\w+`)
```

## Integration Points

### Config (`internal/config/config.go`)

```go
type Project struct {
    // ...existing fields...
    LinkedIn  StringList `toml:"linkedin"`
}

func (p Project) Sources() []SourceEntry {
    return []SourceEntry{
        // ...existing...
        {"linkedin", p.LinkedIn},
    }
}
```

### Tokens (`internal/fetch/fetch.go`)

```go
type Tokens struct {
    GitHub    string
    Plausible string
    YouTube   string
    LinkedIn  string
}

func TokensFromEnv() Tokens {
    return Tokens{
        // ...existing...
        LinkedIn: os.Getenv("LINKEDIN_TOKEN"),
    }
}
```

### Fetch function (`internal/fetch/fetch.go`)

```go
func LinkedIn(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
    if tokens.LinkedIn == "" {
        return []Result{{Source: "linkedin", Skipped: "LINKEDIN_TOKEN not set"}}, nil
    }
    client := &http.Client{Timeout: 30 * time.Second}
    return Source(ctx, cfg, tokens, "linkedin", opts, func(id string, p config.Project) []source.Source {
        var sources []source.Source
        for _, target := range p.LinkedIn {
            sources = append(sources, &source.LinkedIn{Client: client, Token: tokens.LinkedIn, Target: target})
        }
        return sources
    })
}
```

Also add to `All()`:

```go
if tokens.LinkedIn != "" {
    for _, target := range proj.LinkedIn {
        jobs = append(jobs, fetchJob{sourceName: "linkedin", projectID: id, src: &source.LinkedIn{Client: client, Token: tokens.LinkedIn, Target: target}})
    }
}
```

### CLI command (`cmd/velocirepo/cmd/fetch.go`)

```go
var fetchSources = []fetchSourceDef{
    // ...existing...
    {"fetch-linkedin", "Fetch LinkedIn post metrics and content", fetch.LinkedIn},
}
```

### MCP tool (`internal/mcp/server.go`)

```go
s.AddTool(fetchLinkedInTool(), h.handleFetchLinkedIn)

func fetchLinkedInTool() mcp.Tool {
    return mcp.NewTool("fetch_linkedin",
        mcp.WithDescription("Fetch LinkedIn post metrics (impressions, likes, comments, shares) and content index. Requires LINKEDIN_TOKEN."),
        mcp.WithString("project", mcp.Description("Fetch only this project ID")),
        mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
        mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
    )
}
```

### Source dir registry (`internal/config/config.go`)

```go
func SourceDirNames() []string {
    return []string{
        // ...existing...
        "metrics/linkedin",
        "content/linkedin",
    }
}
```

### Export source matching (`internal/store/export.go`)

```go
func matchesSource(table, source string) bool {
    switch source {
    // ...existing...
    case "linkedin":
        return table == "metrics" || table == "content"
    }
}
```

## Staleness Strategy

LinkedIn behaves like YouTube for metrics: it re-fetches stats for ALL posts on every run (post engagement grows over time). The `StartDate`/`EndDate` logic from `resolveStartDate` still applies to the metrics records — if today's file already exists, the fetch is skipped.

For content, the `ContentProvider` path always writes (upsert by id), so new posts are added and existing posts get updated titles/descriptions if edited.

## Test Strategy

Tests use `httptest` with canned responses (same pattern as all other sources):

```go
func TestLinkedInFetchOrganization(t *testing.T) {
    mux := http.NewServeMux()

    mux.HandleFunc("/rest/posts", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(postsResponse{
            Elements: []linkedinPost{
                {
                    ID:         "urn:li:share:7123456789012345678",
                    Author:     "urn:li:organization:12345678",
                    CreatedAt:  1719792000000,
                    Commentary: "Excited to announce our new release! #opensource",
                    Visibility: "PUBLIC",
                    Lifecycle:  "PUBLISHED",
                },
            },
            Paging: struct {
                Count int `json:"count"`
                Start int `json:"start"`
                Total int `json:"total"`
            }{Count: 100, Start: 0, Total: 1},
        })
    })

    mux.HandleFunc("/rest/organizationalEntityShareStatistics", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(statsResponse{
            Elements: []statsElement{
                {
                    Stats: shareStats{ImpressionCount: 4521, LikeCount: 85, CommentCount: 14, ShareCount: 12, ClickCount: 234},
                    Share: "urn:li:share:7123456789012345678",
                },
            },
        })
    })

    mux.HandleFunc("/rest/networkSizes/", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(followerResponse{FirstDegreeSize: 15420})
    })

    srv := httptest.NewServer(mux)
    defer srv.Close()

    li := &LinkedIn{
        Client:  srv.Client(),
        Token:   "test-token",
        Target:  "urn:li:organization:12345678",
        BaseURL: srv.URL,
    }

    records, err := li.Fetch(context.Background(), FetchOptions{
        ProjectID: "my-project",
        StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
        EndDate:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
    })
    if err != nil {
        t.Fatal(err)
    }

    // 5 per-post metrics + 1 follower count = 6
    if len(records) != 6 {
        t.Fatalf("expected 6 records, got %d", len(records))
    }

    entries := li.ContentEntries()
    if len(entries) != 1 {
        t.Fatalf("expected 1 content entry, got %d", len(entries))
    }
    if entries[0].ID != "urn:li:share:7123456789012345678" {
        t.Errorf("unexpected ID: %s", entries[0].ID)
    }
    if entries[0].Tags[0] != "opensource" {
        t.Errorf("expected tag 'opensource', got %v", entries[0].Tags)
    }
}
```

## Auth Command

LinkedIn tokens expire after ~60 days, so manual token management is impractical. A `velocirepo auth linkedin` command automates the OAuth 2.0 flow.

### Design

- **User provides their own app credentials** — no secrets baked into the binary. On first run, the command prompts for client ID and client secret, then stores them in `.env` alongside the resulting token.
- **Local callback server** — the command spins up a temporary HTTP server on `localhost:<port>`, opens the browser to LinkedIn's authorization URL, catches the redirect with the authorization code, exchanges it for an access token, and writes `LINKEDIN_TOKEN=...` to `.env`.
- **Generalizable** — the command structure (`velocirepo auth <provider>`) supports future OAuth providers (e.g., Instagram) without new top-level commands.

### Flow

```
$ velocirepo auth linkedin

LinkedIn OAuth Setup
────────────────────
1. Create a LinkedIn Developer App at https://www.linkedin.com/developers/apps/new
2. Request the "Community Management API" product in the Products tab
3. Add http://localhost:9876/callback as an Authorized Redirect URL in the Auth tab

Client ID: <user pastes>
Client Secret: <user pastes>

Opening browser for authorization...
Waiting for callback...

✓ Token saved to .env (expires in 60 days)
```

### Implementation

```go
// cmd/velocirepo/cmd/auth.go

func authCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "auth <provider>",
        Short: "Authenticate with an OAuth provider",
    }
    cmd.AddCommand(authLinkedInCmd())
    return cmd
}

func authLinkedInCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "linkedin",
        Short: "Authenticate with LinkedIn (OAuth 2.0)",
        RunE:  runAuthLinkedIn,
    }
}
```

```go
// internal/auth/oauth.go

type OAuthFlow struct {
    AuthURL      string
    TokenURL     string
    ClientID     string
    ClientSecret string
    RedirectURI  string
    Scopes       []string
}

func (f *OAuthFlow) Run(ctx context.Context) (string, error) {
    // 1. Start local HTTP server on RedirectURI port
    // 2. Build authorization URL with state param
    // 3. Open browser (exec.Command for open/xdg-open)
    // 4. Wait for callback, extract code
    // 5. Exchange code for token via POST to TokenURL
    // 6. Return access_token
}
```

LinkedIn-specific config:

```go
flow := &auth.OAuthFlow{
    AuthURL:      "https://www.linkedin.com/oauth/v2/authorization",
    TokenURL:     "https://www.linkedin.com/oauth/v2/accessToken",
    ClientID:     clientID,
    ClientSecret: clientSecret,
    RedirectURI:  "http://localhost:9876/callback",
    Scopes:       []string{"r_organization_social"},
}
```

### Token Refresh

LinkedIn access tokens last ~60 days. Options for handling expiry:

1. **Detect 401 at fetch time** — print a clear message: "LinkedIn token expired. Run `velocirepo auth linkedin` to re-authenticate."
2. **Store expiry timestamp** — save `LINKEDIN_TOKEN_EXPIRES=<epoch>` in `.env` and warn proactively before fetching if token is expired or near expiry.

Start with option 1 (simpler); add option 2 later if needed.

### `.env` Management

The auth command reads/updates the `.env` file next to `velocirepo.toml`:

- If `LINKEDIN_CLIENT_ID` / `LINKEDIN_CLIENT_SECRET` already exist in `.env`, skip the prompts
- Always overwrite `LINKEDIN_TOKEN` with the fresh token
- Optionally store `LINKEDIN_TOKEN_EXPIRES` for proactive warnings

## Limitations & Edge Cases

1. **Token expiry**: Tokens expire after ~60 days. The fetch will fail with a 401 — the error message should clearly suggest re-authenticating.

2. **Rate limits**: 100 requests/day is tight. With pagination (1 req per 100 posts) + stats batching (1 req per 20 posts) + 1 follower req, a page with 100 posts costs ~7 requests. This allows ~14 organizations per day, which is fine for typical velocirepo usage.

3. **Personal vs organization**: Personal profiles use slightly different endpoints (`/rest/posts` works for both, but stats may require different permissions). The initial implementation targets organizations; personal profile support can follow.

4. **Post types**: LinkedIn has shares (simple posts), articles (long-form), documents, videos, and polls. All come through the same `/rest/posts` endpoint — the `content` field distinguishes them.

5. **Deleted/unpublished posts**: The API only returns `PUBLISHED` posts. If a post is deleted between fetches, it won't appear in the next fetch but remains in `posts.jsonl` (upsert never removes). This is acceptable — the content file reflects "all posts ever seen."

## Dependency Map

```
cmd/velocirepo/cmd/fetch.go
  └─ {"fetch-linkedin", ..., fetch.LinkedIn}

internal/fetch/fetch.go
  ├─ LinkedIn()           → calls Source() with linkedin source factory
  └─ All()               → adds linkedin jobs when LINKEDIN_TOKEN set

internal/source/linkedin.go (new)
  ├─ LinkedIn struct      → implements Source + ContentProvider
  ├─ Fetch()             → returns []Record (engagement metrics)
  ├─ ContentEntries()    → returns []ContentEntry (post listings)
  └─ ContentFilename()   → "posts.jsonl"

internal/config/config.go
  └─ Project.LinkedIn    → StringList `toml:"linkedin"`

internal/mcp/server.go
  └─ fetchLinkedInTool() + handleFetchLinkedIn
```

## TODO

- [x] Implement `internal/auth/oauth.go`:
  - [x] `OAuthFlow` struct with `AuthURL`, `TokenURL`, `ClientID`, `ClientSecret`, `RedirectURI`, `Scopes`
  - [x] `Run()` method: start local server, open browser, wait for callback, exchange code for token
  - [x] Helper to update `.env` file (read, upsert key, write)
- [x] Add `cmd/velocirepo/cmd/auth.go`:
  - [x] `auth` parent command
  - [x] `auth linkedin` subcommand: prompt for client ID/secret (if not in `.env`), run OAuth flow, save token
- [x] Register `authCmd()` in root command
- [x] Add `LinkedIn StringList` field to `config.Project`
- [x] Add `{"linkedin", p.LinkedIn}` to `Project.Sources()`
- [x] Add `"metrics/linkedin"` and `"content/linkedin"` to `SourceDirNames()`
- [x] Add `LinkedIn string` to `fetch.Tokens` and `TokensFromEnv()`
- [x] Implement `internal/source/linkedin.go`:
  - [x] `LinkedIn` struct with `Client`, `Token`, `Target`, `BaseURL`, `contentEntries`
  - [x] `Fetch()` → fetch posts, build content entries, fetch stats, fetch followers
  - [x] `fetchPosts()` with pagination
  - [x] `fetchPostStats()` with batching (20 per request)
  - [x] `fetchFollowerCount()` for organizations
  - [x] `postToContentEntry()` helper
  - [x] `get()` helper with Bearer auth + LinkedIn-Version header
  - [x] API response type structs
- [x] Add `internal/source/linkedin_test.go` with httptest canned responses
- [x] Add `fetch.LinkedIn()` function in `internal/fetch/fetch.go`
- [x] Add linkedin jobs to `fetch.All()`
- [x] Add `{"fetch-linkedin", ..., fetch.LinkedIn}` to `fetchSources` in cmd
- [x] Add `fetchLinkedInTool()` + `handleFetchLinkedIn` to MCP server
- [x] Add `"linkedin"` case to `matchesSource()` in `internal/store/export.go`
- [x] Update MCP query tool description to mention linkedin metrics
- [x] Add `"linkedin"` to `handleAddProject` and `handleUpdateProject` string fields
