package data

type DatabaseInfo struct {
	name string
	ttl  int //s
}

func NewDataBaseInfo(name string, ttl int) DatabaseInfo {
	return DatabaseInfo{
		name: name,
		ttl:  ttl,
	}
}

func (d *DatabaseInfo) GetTTL() int {
	return d.ttl
}
