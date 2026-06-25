package views

import (
	"os"
	"path/filepath"
	"strings"
)

type View struct {
	Name string
	Dir  string
}

func Discover(viewsDir string) ([]View, error) {
	var views []View

	entries, err := os.ReadDir(viewsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "_output" || name == "_data" {
			continue
		}
		dir := filepath.Join(viewsDir, name)
		discoverRecursive(dir, name, &views)
	}

	return views, nil
}

func discoverRecursive(dir, prefix string, views *[]View) {
	renderScript := filepath.Join(dir, "render.sh")
	if _, err := os.Stat(renderScript); err == nil {
		*views = append(*views, View{Name: prefix, Dir: dir})
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "_output" || name == "_data" {
			continue
		}
		subdir := filepath.Join(dir, name)
		discoverRecursive(subdir, prefix+"/"+name, views)
	}
}

func FindView(views []View, name string) (View, bool) {
	for _, v := range views {
		if v.Name == name {
			return v, true
		}
	}
	return View{}, false
}

func FindViews(views []View, name string) []View {
	for _, v := range views {
		if v.Name == name {
			return []View{v}
		}
	}
	prefix := strings.TrimSuffix(name, "/") + "/"
	var matched []View
	for _, v := range views {
		if strings.HasPrefix(v.Name, prefix) {
			matched = append(matched, v)
		}
	}
	return matched
}
