// SPDX-License-Identifier: Apache-2.0

package cargo

import (
	"path/filepath"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

type Mod struct {
	metadata      plugin.Metadata
	rootModule    *meta.Package
	command       *helper.Cmd
	cargoMetadata Metadata
}

func New() *Mod {
	return &Mod{
		metadata: plugin.Metadata{
			Name:       "Cargo Modules",
			Slug:       "cargo",
			Manifest:   []string{CargoTomlFile},
			ModulePath: []string{"vendor"},
		},
	}
}
func (m *Mod) GetMetadata() plugin.Metadata {
	return m.metadata
}

func (m *Mod) SetRootModule(path string) error {
	module, err := m.getRootModule(path)
	if err != nil {
		return err
	}

	m.rootModule = &module
	return nil
}

func (m *Mod) GetVersion() (string, error) {
	if err := m.buildCmd(VersionCmd, "."); err != nil {
		return "", err
	}

	return m.command.Output()
}

func (m *Mod) GetRootModule(path string) (*meta.Package, error) {
	if err := m.SetRootModule(path); err != nil {
		return nil, err
	}

	return m.rootModule, nil
}

func (m *Mod) ListUsedModules(path string) ([]meta.Package, error) {
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

	modules := convertMetadataToModulesList(meta.Packages)
	collection = append(collection, modules...)

	return collection, nil
}

func (m *Mod) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	modules, err := m.ListUsedModules(path)
	if err != nil {
		return nil, err
	}

	meta, err := m.getCargoMetadata(path)
	if err != nil {
		return nil, err
	}

	addDepthModules(modules, meta.Packages)

	return modules, nil
}

func (m *Mod) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if helper.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

func (m *Mod) HasModulesInstalled(path string) error {
	if helper.Exists(filepath.Join(path, CargoLockFile)) {
		return nil
	}
	return errDependenciesNotFound
}
