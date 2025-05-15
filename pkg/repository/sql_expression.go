package repository

import sq "github.com/Masterminds/squirrel"

type expression interface {
	toExp() (sq.Sqlizer, error)
}

type binaryExpression interface {
	toExp(key string) (sq.Sqlizer, error)
}

func (a And) toExp() (sq.Sqlizer, error) {
	exps := make(sq.And, 0)
	for _, item := range a {
		if f, ok := item.(expression); ok {
			exp, err := f.toExp()
			if err != nil {
				return nil, nil
			}
			exps = append(exps, exp)
		}
	}
	return exps, nil
}

func (o Or) toExp() (sq.Sqlizer, error) {
	exps := make(sq.Or, 0)
	for _, item := range o {
		if f, ok := item.(expression); ok {
			exp, err := f.toExp()
			if err != nil {
				return nil, nil
			}
			exps = append(exps, exp)
		}
	}
	return exps, nil
}

func (b *Binary) toExp() (sq.Sqlizer, error) {
	if o, ok := b.Op.(binaryExpression); ok {
		return o.toExp(b.Key)
	}
	return nil, nil
}

func (e *Eq) toExp(key string) (sq.Sqlizer, error) {
	return sq.Eq{key: e.Value}, nil
}

func (n *Ne) toExp(key string) (sq.Sqlizer, error) {
	return sq.NotEq{key: n.Value}, nil
}

func (g *Gt) toExp(key string) (sq.Sqlizer, error) {
	return sq.Gt{key: g.Value}, nil
}

func (g *Gte) toExp(key string) (sq.Sqlizer, error) {
	return sq.GtOrEq{key: g.Value}, nil
}

func (l *Lt) toExp(key string) (sq.Sqlizer, error) {
	return sq.Lt{key: l.Value}, nil
}

func (l *Lte) toExp(key string) (sq.Sqlizer, error) {
	return sq.LtOrEq{key: l.Value}, nil
}

func (s *Start) toExp(key string) (sq.Sqlizer, error) {
	return sq.Like{key: string(*s) + "%"}, nil
}

func (e *End) toExp(key string) (sq.Sqlizer, error) {
	return sq.Like{key: "%" + string(*e)}, nil
}

func (c *Contains) toExp(key string) (sq.Sqlizer, error) {
	return sq.Like{key: "%" + string(*c) + "%"}, nil
}

func (r *Regexp) toExp(key string) (sq.Sqlizer, error) {
	return sq.Expr(key+" REGEXP ?", *r), nil
}

func (e *Exist) toExp(key string) (sq.Sqlizer, error) {
	if *e {
		return sq.Expr(key + " IS NOT NULL"), nil
	}
	return sq.Expr(key + " IS NULL"), nil
}

func (i *In) toExp(key string) (sq.Sqlizer, error) {
	return sq.Eq{key: *i}, nil
}

func (n *Nin) toExp(key string) (sq.Sqlizer, error) {
	return sq.NotEq{key: *n}, nil
}
