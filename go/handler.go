// SPDX-License-Identifier: Apache-2.0

package gomod

import (
	"bytes"
	"path/filepath"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

// New ...
func New() *GoMod {
	return &GoMod{
		metadata: plugin.Metadata{
			Name:     "Go Modules",
			Slug:     "go-mod",
			Manifest: []string{"go.mod"},
		},
	}
}

// GetMetadata ...
func (m *GoMod) GetMetadata() plugin.Metadata {
	return m.metadata
}

// SetRootModule ...
func (m *GoMod) SetRootModule(path string) error {
	module, err := m.getModule(path)
	if err != nil {
		return err
	}

	m.rootModule = &module

	return nil
}

// IsValid ...
func (m *GoMod) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if helper.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

// HasModulesInstalled ...
func (m *GoMod) HasModulesInstalled(path string) error {
	// we dont need to validate if packages are installed as process to
	// read dependencies will download them
	return nil
}

// GetVersion...
func (m *GoMod) GetVersion() (string, error) {
	if err := m.buildCmd(VersionCmd, "."); err != nil {
		return "", err
	}

	return m.command.Output()
}

// GetRootModule...
func (m *GoMod) GetRootModule(path string) (*meta.Package, error) {
	if m.rootModule == nil {
		module, err := m.getModule(path)
		if err != nil {
			return nil, err
		}

		m.rootModule = &module
	}

	return m.rootModule, nil
}

// ListUsedModules...
func (m *GoMod) ListUsedModules(path string) ([]meta.Package, error) {
	if err := m.buildCmd(ModulesCmd, path); err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	if err := m.command.Execute(buffer); err != nil {
		return nil, err
	}
	defer buffer.Reset()

	mainModule, err := m.GetRootModule(path)
	if err != nil {
		return nil, err
	}

	modules := []meta.Package{}
	if err := NewDecoder(buffer).ConvertJSONReaderToModules(mainModule.Path, &modules); err != nil {
		return nil, err
	}

	return modules, nil
}

// ListModulesWithDeps ...
func (m *GoMod) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	modules, err := m.ListUsedModules(path)
	if err != nil {
		return nil, err
	}

	if err := m.buildCmd(GraphModuleCmd, path); err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	if err := m.command.Execute(buffer); err != nil {
		return nil, err
	}
	defer buffer.Reset()

	if err := NewDecoder(buffer).ConvertPlainReaderToModules(modules); err != nil {
		return nil, err
	}

	return modules, nil
}

func (m *GoMod) getModule(path string) (meta.Package, error) {
	if err := m.buildCmd(RootModuleCmd, path); err != nil {
		return meta.Package{}, err
	}

	buffer := new(bytes.Buffer)
	if err := m.command.Execute(buffer); err != nil {
		return meta.Package{}, err
	}
	defer buffer.Reset()

	module := meta.Package{}
	if err := NewDecoder(buffer).ConvertJSONReaderToSingleModule(&module); err != nil {
		return meta.Package{}, err
	}

	if module.Path == "" {
		return meta.Package{}, errFailedToConvertModules
	}

	return module, nil
}

func (m *GoMod) buildCmd(cmd command, path string) error {
	cmdArgs := cmd.Parse()
	if cmdArgs[0] != "go" {
		return errNoGoCommand
	}

	command := helper.NewCmd(helper.CmdOptions{
		Name:      cmdArgs[0],
		Args:      cmdArgs[1:],
		Directory: path,
	})

	m.command = command

	return command.Build()
}
