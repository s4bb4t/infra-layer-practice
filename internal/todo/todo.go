// Package todo marks code the mentee has to write.
//
// A stub returns todo.Step("03"). Contract tests turn such errors into SKIP,
// so `make check` stays green on main while a step is not done yet.
// A step is done when its tests show no SKIP: `go test -v ... | grep SKIP` is empty.
package todo

import (
	"errors"
	"fmt"
)

var ErrNotImplemented = errors.New("not implemented")

// Step returns an error that says which task to open.
func Step(step string) error {
	return fmt.Errorf("%w: see tasks/%s-*.md", ErrNotImplemented, step)
}
