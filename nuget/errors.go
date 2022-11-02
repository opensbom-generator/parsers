// SPDX-License-Identifier: Apache-2.0

package nuget

import (
	"errors"
)

type errType error

var errDependenciesNotFound errType = errors.New("unable to generate SPDX file, no modules or packages found. Please install them before running spdx-sbom-generator, e.g.: `dotnet restore`")
var errNoDotnetCommand errType = errors.New("no dotnet command")
var errNoDependencyCache errType = errors.New("local dependency cache not found")
var errFailedToConvertModules errType = errors.New("failed to convert modules")
