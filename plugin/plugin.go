// SPDX-License-Identifier: Apache-2.0

package plugin

import "github.com/opensbom-generator/parsers/meta"

type Plugin interface {
	SetRootModule(path string) error
	GetVersion() (string, error)
	GetMetadata() Metadata
	GetRootModule(path string) (*meta.Package, error)
	ListUsedModules(path string) ([]meta.Package, error)
	ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error)
	IsValid(path string) bool
	HasModulesInstalled(path string) error
}

// Metadata
type Metadata struct {
	Name       string
	Slug       string
	Manifest   []string
	ModulePath []string
}
