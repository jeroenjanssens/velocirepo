package sourceinfo

// Category is the date-partitioned data category for a source.
type Category string

const (
	CategoryEvents  Category = "events"
	CategoryMetrics Category = "metrics"
)

// Descriptor captures source metadata shared by config, CLI, MCP, fetch, and
// store path handling.
type Descriptor struct {
	Name        string
	DisplayName string
	Category    Category
	ContentDir  string
}

var descriptors = []Descriptor{
	{Name: "github", DisplayName: "GitHub", Category: CategoryEvents},
	{Name: "github-traffic", DisplayName: "GitHub Traffic", Category: CategoryMetrics},
	{Name: "pypi", DisplayName: "PyPI", Category: CategoryMetrics},
	{Name: "cran", DisplayName: "CRAN", Category: CategoryMetrics},
	{Name: "homebrew", DisplayName: "Homebrew", Category: CategoryMetrics},
	{Name: "plausible", DisplayName: "Plausible", Category: CategoryMetrics},
	{Name: "openvsx", DisplayName: "OpenVSX", Category: CategoryMetrics},
	{Name: "youtube", DisplayName: "YouTube", Category: CategoryMetrics, ContentDir: "content/youtube"},
	{Name: "linkedin", DisplayName: "LinkedIn", Category: CategoryMetrics, ContentDir: "content/linkedin"},
}

var byName = func() map[string]Descriptor {
	m := make(map[string]Descriptor, len(descriptors))
	for _, d := range descriptors {
		m[d.Name] = d
	}
	return m
}()

func All() []Descriptor {
	out := make([]Descriptor, len(descriptors))
	copy(out, descriptors)
	return out
}

func Get(name string) (Descriptor, bool) {
	d, ok := byName[name]
	return d, ok
}

func IsEvent(name string) bool {
	d, ok := Get(name)
	return ok && d.Category == CategoryEvents
}

func CategoryDir(name string) string {
	if d, ok := Get(name); ok {
		return string(d.Category)
	}
	return string(CategoryMetrics)
}

func DataDirPath(name string) string {
	return CategoryDir(name) + "/" + name
}

func DataDirPaths() []string {
	paths := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		paths = append(paths, DataDirPath(d.Name))
		if d.ContentDir != "" {
			paths = append(paths, d.ContentDir)
		}
	}
	return paths
}

func EventNames() []string {
	var names []string
	for _, d := range descriptors {
		if d.Category == CategoryEvents {
			names = append(names, d.Name)
		}
	}
	return names
}

func MetricNames() []string {
	var names []string
	for _, d := range descriptors {
		if d.Category == CategoryMetrics {
			names = append(names, d.Name)
		}
	}
	return names
}
