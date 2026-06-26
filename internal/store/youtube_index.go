package store

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func readYouTubeIndex(path string) ([]source.YouTubeIndexEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []source.YouTubeIndexEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var e source.YouTubeIndexEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}
