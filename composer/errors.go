// SPDX-License-Identifier: Apache-2.0

package composer

import (
	"errors"
)

type errType error

var (
	errDependenciesNotFound     errType = errors.New("no dependencies installed. Please install Modules before running spdx-sbom-generator, e.g.: `composer install`")
	errNoComposerCommand        errType = errors.New("no Composer command")
	errFailedToReadComposerFile errType = errors.New("failed to read composer lock files")
	errFailedToShowComposerTree errType = errors.New("failed to show composer tree")
	errRootProject              errType = errors.New("failed to read root project info")
)
