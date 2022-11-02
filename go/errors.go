// SPDX-License-Identifier: Apache-2.0

package gomod

import (
	"errors"
)

type errType error

var (
	errNoGoCommand            errType = errors.New("no Golang command")
	errFailedToConvertModules errType = errors.New("failed to convert modules")
)
