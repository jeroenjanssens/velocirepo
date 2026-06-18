package source

func filterByMetric(records []Record, metric string) []Record {
	var filtered []Record
	for _, r := range records {
		if r.Metric == metric {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
