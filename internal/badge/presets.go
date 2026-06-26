package badge

type Preset struct {
	Label string
	Query string
	Color string
}

var Presets = map[string]Preset{
	"stars":     {"stars", "SELECT COUNT(*) AS value FROM events WHERE type = 'star'", "#007ec6"},
	"forks":     {"forks", "SELECT COUNT(*) AS value FROM events WHERE type = 'fork'", "#007ec6"},
	"downloads": {"downloads", "SELECT MAX(value) AS value FROM metrics WHERE metric = 'downloads' OR metric = 'total_downloads'", "#44cc11"},
	"pageviews": {"pageviews", "SELECT SUM(value) AS value FROM metrics WHERE metric = 'pageviews'", "#44cc11"},
}
