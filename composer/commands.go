// SPDX-License-Identifier: Apache-2.0

package composer

import (
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
)

type command string

var (
	VersionCmd           command = "composer --version"
	ShowModulesCmd       command = "composer show -t -f json"
	projectInfoCmd       command = "composer show -s -f json"
	ComposerLockFileName         = "composer.lock"
	ComposerJSONFileName         = "composer.json"
	PackageJSON                  = "package.json"
	ComposerVendorFolder         = "vendor"
)

// Parse ...
func (c command) Parse() []string {
	cmd := strings.TrimSpace(string(c))
	return strings.Fields(cmd)
}

func (m *Composer) buildCmd(cmd command, path string) error {
	cmdArgs := cmd.Parse()
	if cmdArgs[0] != "composer" {
		return errNoComposerCommand
	}

	command := helper.NewCmd(helper.CmdOptions{
		Name:      cmdArgs[0],
		Args:      cmdArgs[1:],
		Directory: path,
	})

	m.command = command

	return command.Build()
}
