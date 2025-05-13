package generic

type NameFilterFunc struct {
	Eq         string
	In         []string
	Contains   string
	StartsWith string
	EndsWith   string
}
