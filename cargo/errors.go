// SPDX-License-Identifier: Apache-2.0

package cargo

import (
	"errors"
)

type errType error

var errNoCmd errType = errors.New("no cargo command")
var errDependenciesNotFound errType = errors.New("unable to generate SPDX file, no modules or vendors found. Please install them before running spdx-sbom-generator, e.g.: `cargo build`")
var errBuildlingModuleDependencies errType = errors.New("error building modules dependencies")
var errNoCargoCommand errType = errors.New("no Cargo command")
var erroRootPackageInformation errType = errors.New("failed to read root folder information. Please verify you can run `cargo pkgid`")
var errFailedToConvertModules errType = errors.New("failed to convert modules")
