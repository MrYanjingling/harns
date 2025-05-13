package strutil

func lastSlash(s string) int {
	i := len(s) - 1
	for i >= 0 && s[i] != '/' {
		i--
	}
	return i
}

// Split copied from path.Split but dir without end slash
func Split(path string) (dir, file string) {
	i := lastSlash(path)
	return path[:i], path[i+1:]
}
