// SPDX-License-Identifier: Apache-2.0

package cargo

import (
	"path/filepath"

	"github.com/opensbom-generator/parsers/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

type mod struct {
	metadata      plugin.Metadata
	rootModule    *meta.Package
	command       *helper.Cmd
	cargoMetadata CargoMetadata
}

func New() *mod {
	return &mod{
		metadata: plugin.Metadata{
			Name:       "Cargo Modules",
			Slug:       "cargo",
			Manifest:   []string{CargoTomlFile},
			ModulePath: []string{"vendor"},
		},
	}
}
func (m *mod) GetMetadata() plugin.Metadata {
	return m.metadata
}

func (m *mod) SetRootModule(path string) error {
	module, err := m.getRootModule(path)
	if err != nil {
		return err
	}

	m.rootModule = &module
	return nil
}

func (m *mod) GetVersion() (string, error) {
	if err := m.buildCmd(VersionCmd, "."); err != nil {
		return "", err
	}

	return m.command.Output()
}

func (m *mod) GetRootModule(path string) (*meta.Package, error) {
	if err := m.SetRootModule(path); err != nil {
		return nil, err
	}

	return m.rootModule, nil
}

func (m *mod) ListUsedModules(path string) ([]meta.Package, error) {
	var collection []meta.Package

	rootModule, err := m.GetRootModule(path)
	if err != nil {
		return nil, err
	}

	collection = append(collection, *rootModule)
	meta, err := m.getCargoMetadata(path)
	if err != nil {
		return nil, err
	}
	modules, err := convertMetadataToModulesList(meta.Packages)
	if err != nil {
		return nil, err
	}

	collection = append(collection, modules...)

	return collection, nil
}

func (m *mod) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	modules, err := m.ListUsedModules(path)
	if err != nil {
		return nil, err
	}

	meta, err := m.getCargoMetadata(path)
	if err != nil {
		return nil, err
	}

	err = addDepthModules(modules, meta.Packages)
	if err != nil {
		return nil, err
	}

	return modules, nil
}

func (m *mod) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if helper.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

func (m *mod) HasModulesInstalled(path string) error {
	if helper.Exists(filepath.Join(path, CargoLockFile)) {
		return nil
	}
	return errDependenciesNotFound
}
