package cli

import (
	"errors"

	"github.com/tamnd/archwiki-cli/archwiki"
)

func isNotFound(err error) bool {
	return errors.Is(err, archwiki.ErrNotFound)
}
