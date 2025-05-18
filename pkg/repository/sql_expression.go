package repository

import sq "github.com/Masterminds/squirrel"

type expression interface {
	toSql(d dialect) (sq.Sqlizer, error)
}

var (
	_ expression = (And)(nil)
	_ expression = (Or)(nil)
	_ expression = (*Binary)(nil)
)

type condition interface {
	toSql(d dialect, key string) (sq.Sqlizer, error)
}

var (
	_ condition = (*Eq)(nil)
	_ condition = (*Ne)(nil)
	_ condition = (*Gte)(nil)
	_ condition = (*Gt)(nil)
	_ condition = (*Lte)(nil)
	_ condition = (*Lt)(nil)
	_ condition = (*In)(nil)
	_ condition = (*Nin)(nil)
	_ condition = (*Start)(nil)
	_ condition = (*End)(nil)
	_ condition = (*Contains)(nil)
	_ condition = (*Regexp)(nil)
	_ condition = (*Exist)(nil)
)

func (a And) toSql(d dialect) (sq.Sqlizer, error) {
	exps := make(sq.And, 0)
	for _, item := range a {
		if f, ok := item.(expression); ok {
			exp, err := f.toSql(d)
			if err != nil {
				return nil, nil
			}
			exps = append(exps, exp)
		}
	}
	return exps, nil
}

func (o Or) toSql(d dialect) (sq.Sqlizer, error) {
	exps := make(sq.Or, 0)
	for _, item := range o {
		if f, ok := item.(expression); ok {
			exp, err := f.toSql(d)
			if err != nil {
				return nil, nil
			}
			exps = append(exps, exp)
		}
	}
	return exps, nil
}

func (b *Binary) toSql(d dialect) (sq.Sqlizer, error) {
	if o, ok := b.Op.(condition); ok {
		return o.toSql(d, d.parseJsonKey(b.Key))
	}
	return nil, nil
}

func (e *Eq) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Eq{key: e.Value}, nil
}

func (n *Ne) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.NotEq{key: n.Value}, nil
}

func (g *Gt) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Gt{key: g.Value}, nil
}

func (g *Gte) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.GtOrEq{key: g.Value}, nil
}

func (l *Lt) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Lt{key: l.Value}, nil
}

func (l *Lte) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.LtOrEq{key: l.Value}, nil
}

func (s *Start) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Like{key: string(*s) + "%"}, nil
}

func (e *End) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Like{key: "%" + string(*e)}, nil
}

func (c *Contains) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Like{key: "%" + string(*c) + "%"}, nil
}

func (r *Regexp) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Expr(key+" REGEXP ?", *r), nil
}

func (e *Exist) toSql(d dialect, key string) (sq.Sqlizer, error) {
	if *e {
		return sq.Expr(key + " IS NOT NULL"), nil
	}
	return sq.Expr(key + " IS NULL"), nil
}

func (i *In) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Eq{key: *i}, nil
}

func (n *Nin) toSql(d dialect, key string) (sq.Sqlizer, error) {
	return sq.NotEq{key: *n}, nil
}
