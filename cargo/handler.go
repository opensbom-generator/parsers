// SPDX-License-Identifier: Apache-2.0

package cargo

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
	"sigs.k8s.io/release-utils/command"
	"sigs.k8s.io/release-utils/util"
)

type Mod struct {
	metadata      plugin.Metadata
	rootModule    *meta.Package
	cargoMetadata *Metadata
	impl          cargoImplementation
}

func New() *Mod {
	return &Mod{
		metadata: plugin.Metadata{
			Name:       "Cargo Modules",
			Slug:       "cargo",
			Manifest:   []string{tomlFileName},
			ModulePath: []string{"vendor"},
		},
		impl: &defaultImplementation{},
	}
}

func (m *Mod) SetImplementation(impl cargoImplementation) {
	m.impl = impl
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
	output, err := command.NewWithWorkDir(".", "cargo", "--version").RunSilentSuccessOutput()
	if err != nil {
		return "", fmt.Errorf("getting cargo version: %w", err)
	}

	return strings.TrimPrefix(output.OutputTrimNL(), "cargo "), nil
}

func (m *Mod) GetRootModule(path string) (*meta.Package, error) {
	if m.rootModule != nil {
		return m.rootModule, nil
	}

	md, err := m.impl.GetCargoMetadataIfNeeded(m, path)
	if err != nil {
		return nil, fmt.Errorf("getting cargo metadata: %w", err)
	}

	rootName, err := m.impl.GetRootProjectName(path)
	if err != nil {
		return nil, fmt.Errorf("getting project name: %w", err)
	}

	cargoPackage := md.GetPackageByName(rootName)
	metaPackage := m.impl.ConvertCargoPackageToMetaPackage(cargoPackage)

	return &metaPackage, nil
}

// ListUsedModules returns the firs tier dependencies of the module
func (m *Mod) ListUsedModules(path string) ([]meta.Package, error) {
	md, err := m.impl.GetCargoMetadataIfNeeded(m, path)
	if err != nil {
		return nil, fmt.Errorf("getting cargo metadata: %w", err)
	}

	mod, err := m.impl.GetRootModule(md, path)
	if err != nil {
		return nil, fmt.Errorf("getting root module: %w", err)
	}

	if err := m.impl.PopulateDependencies(md, &mod, false, nil); err != nil {
		return nil, fmt.Errorf("populating deps of %s: %w", mod.Name, err)
	}

	r := []meta.Package{}
	for _, p := range mod.Packages {
		r = append(r, *p)
	}
	return r, nil
}

func (m *Mod) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	md, err := m.impl.GetCargoMetadataIfNeeded(m, path)
	if err != nil {
		return nil, fmt.Errorf("getting cargo metadata: %w", err)
	}

	mod, err := m.impl.GetRootModule(md, path)
	if err != nil {
		return nil, fmt.Errorf("getting root module: %w", err)
	}

	if err := m.impl.PopulateDependencies(md, &mod, true, nil); err != nil {
		return nil, fmt.Errorf("populating deps of %s: %w", mod.Name, err)
	}

	r := []meta.Package{}
	for _, p := range mod.Packages {
		r = append(r, *p)
	}
	return r, nil
}

func (m *Mod) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if util.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

func (m *Mod) HasModulesInstalled(path string) error {
	if util.Exists(filepath.Join(path, lockFileName)) {
		return nil
	}
	return errors.New("project lockfile not found")
}
