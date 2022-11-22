package errors

import "errors"

// ErrNoOrganizationName is returned by commands that require an organization name to be provided, but none was provided.
var ErrNoOrganizationName = errors.New("You do not have an organization selected, select one using: \n\n\tjsctl config set organization [name]")
