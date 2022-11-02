// SPDX-License-Identifier: Apache-2.0

package pyenv

import (
	"strings"
)

type Command string

// assume each project is using python3 default
const (
	VersionCmd           Command = "{executable}/python -V"                           // generic to check version
	ModulesCmd           Command = "{executable}/python -m pip list -v --format json" // venv is local
	MetadataCmd          Command = "{executable}/python -m pip show {PACKAGE}"
	InstallRootModuleCmd Command = "{executable}/python -m pip install -e .."
)

// Parse ...
func (c Command) Parse() []string {
	cmd := strings.TrimSpace(string(c))
	return strings.Fields(cmd)
}
