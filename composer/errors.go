// SPDX-License-Identifier: Apache-2.0

package composer

import (
	"errors"
)

type errType error

var errDependenciesNotFound = errors.New("no dependencies installed. Please install Modules before running spdx-sbom-generator, e.g.: `composer install`")
var errNoComposerCommand = errors.New("no Composer command")
var errFailedToReadComposerFile errType = errors.New("failed to read composer lock files")
var errFailedToShowComposerTree errType = errors.New("failed to show composer tree")
var errRootProject errType = errors.New("failed to read root project info")
