package source

func splitOwnerRepo(repo string) (string, string) {
	for i, c := range repo {
		if c == '/' {
			return repo[:i], repo[i+1:]
		}
	}
	return "", ""
}
