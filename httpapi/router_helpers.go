package httpapi

import (
	"errors"

	"example.com/grid-fault-locate-service/domain"
)

func notFoundError(path string) error {
	return errors.Join(domain.ErrNotFound, errors.New("api route not found: "+path))
}
