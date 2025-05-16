package repository

import sq "github.com/Masterminds/squirrel"

type expression interface {
	toExp(d dialect) (sq.Sqlizer, error)
}

type binaryExpression interface {
	toExp(d dialect, key string) (sq.Sqlizer, error)
}

func (a And) toExp(d dialect) (sq.Sqlizer, error) {
	exps := make(sq.And, 0)
	for _, item := range a {
		if f, ok := item.(expression); ok {
			exp, err := f.toExp(d)
			if err != nil {
				return nil, nil
			}
			exps = append(exps, exp)
		}
	}
	return exps, nil
}

func (o Or) toExp(d dialect) (sq.Sqlizer, error) {
	exps := make(sq.Or, 0)
	for _, item := range o {
		if f, ok := item.(expression); ok {
			exp, err := f.toExp(d)
			if err != nil {
				return nil, nil
			}
			exps = append(exps, exp)
		}
	}
	return exps, nil
}

func (b *Binary) toExp(d dialect) (sq.Sqlizer, error) {
	if o, ok := b.Op.(binaryExpression); ok {
		return o.toExp(d, d.parseJsonKey(b.Key))
	}
	return nil, nil
}

func (e *Eq) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Eq{key: e.Value}, nil
}

func (n *Ne) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.NotEq{key: n.Value}, nil
}

func (g *Gt) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Gt{key: g.Value}, nil
}

func (g *Gte) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.GtOrEq{key: g.Value}, nil
}

func (l *Lt) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Lt{key: l.Value}, nil
}

func (l *Lte) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.LtOrEq{key: l.Value}, nil
}

func (s *Start) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Like{key: string(*s) + "%"}, nil
}

func (e *End) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Like{key: "%" + string(*e)}, nil
}

func (c *Contains) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Like{key: "%" + string(*c) + "%"}, nil
}

func (r *Regexp) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Expr(key+" REGEXP ?", *r), nil
}

func (e *Exist) toExp(d dialect, key string) (sq.Sqlizer, error) {
	if *e {
		return sq.Expr(key + " IS NOT NULL"), nil
	}
	return sq.Expr(key + " IS NULL"), nil
}

func (i *In) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.Eq{key: *i}, nil
}

func (n *Nin) toExp(d dialect, key string) (sq.Sqlizer, error) {
	return sq.NotEq{key: *n}, nil
}
