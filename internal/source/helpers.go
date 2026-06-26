package source

import "time"

// inDateRange reports whether t is within [start, end] (inclusive on both sides).
func inDateRange(t time.Time, start, end time.Time) bool {
	return !t.Before(start) && t.Before(end.AddDate(0, 0, 1))
}

func splitOwnerRepo(repo string) (string, string) {
	for i, c := range repo {
		if c == '/' {
			return repo[:i], repo[i+1:]
		}
	}
	return "", ""
}
