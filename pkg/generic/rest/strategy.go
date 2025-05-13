package rest

import (
	"context"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic/runtime"
)

type CreateStrategy interface {
	Validate(ctx context.Context, obj runtime.Object) field.ErrorList
}

type UpdateStrategy interface {
	CreateStrategy
	ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList
}

func ValidateFunc(cs CreateStrategy) ValidateObjectFunc {
	return func(ctx context.Context, obj runtime.Object) error {
		if errs := cs.Validate(ctx, obj); len(errs) != 0 {
			return response.ErrInvalidValue(errs.ToAggregate())
		}
		return nil
	}
}

func ValidateUpdateFunc(us UpdateStrategy) ValidateObjectUpdateFunc {
	return func(ctx context.Context, obj, old runtime.Object) error {
		if errs := us.Validate(ctx, obj); len(errs) != 0 {
			return response.ErrInvalidValue(errs.ToAggregate())
		}
		if errs := us.ValidateUpdate(ctx, obj, old); len(errs) != 0 {
			return response.ErrInvalidUpdate(errs.ToAggregate())
		}
		return nil
	}
}
