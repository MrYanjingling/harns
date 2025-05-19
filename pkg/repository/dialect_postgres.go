package repository

type dialectPostgres struct{}

func (d dialectPostgres) buildCreateTable(table string, schema Schema) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (d dialectPostgres) parseJsonKey(key string) string {
	//TODO implement me
	panic("implement me")
}
