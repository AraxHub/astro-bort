package domain

import "errors"

// BusinessError ошибка бизнес-логики, которая уже залогирована в UseCase
type BusinessError struct {
	Err error
}

func (e *BusinessError) Error() string {
	return e.Err.Error()
}

func (e *BusinessError) Unwrap() error {
	return e.Err
}

func WrapBusinessError(err error) error {
	if err == nil {
		return nil
	}
	return &BusinessError{Err: err}
}

func IsBusinessError(err error) bool {
	var businessErr *BusinessError
	return errors.As(err, &businessErr)
}
