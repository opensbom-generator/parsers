// SPDX-License-Identifier: Apache-2.0

package gomod

import (
	"errors"
)

type errType error

var errDependenciesNotFound errType = errors.New("unable to generate SPDX file, no modules or vendors found. Please install them before running spdx-sbom-generator, e.g.: `go mod vendor`")
var errBuildlingModuleDependencies errType = errors.New("error building modules dependencies")
var errNoGoCommand errType = errors.New("'go' command not found")
var errFailedToConvertModules errType = errors.New("failed to convert modules")
var errNoMainModule errType = errors.New("main module not found")
