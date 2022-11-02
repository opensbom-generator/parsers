// SPDX-License-Identifier: Apache-2.0

package pip

import (
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/pip/pipenv"
	"github.com/opensbom-generator/parsers/pip/poetry"
	"github.com/opensbom-generator/parsers/pip/pyenv"

	"github.com/opensbom-generator/parsers/plugin"
)

type PIP struct {
	plugin plugin.Plugin
}

// New ...
func New() *PIP {
	return &PIP{
		plugin: nil,
	}
}

// Get Metadata ...
func (m *PIP) GetMetadata() plugin.Metadata {
	return m.plugin.GetMetadata()
}

// Is Valid ...
func (m *PIP) IsValid(path string) bool {
	if p := pipenv.New(); p.IsValid(path) {
		m.plugin = p
		return true
	}

	if p := poetry.New(); p.IsValid(path) {
		m.plugin = p
		return true
	}

	if p := pyenv.New(); p.IsValid(path) {
		m.plugin = p
		return true
	}

	return false
}

// Has Modules Installed ...
func (m *PIP) HasModulesInstalled(path string) error {
	return m.plugin.HasModulesInstalled(path)
}

// Get Version ...
func (m *PIP) GetVersion() (string, error) {
	return m.plugin.GetVersion()
}

// Set Root Module ...
func (m *PIP) SetRootModule(path string) error {
	return m.plugin.SetRootModule(path)
}

// Get Root Module ...
func (m *PIP) GetRootModule(path string) (*meta.Package, error) {
	return m.plugin.GetRootModule(path)
}

// List Used Modules...
func (m *PIP) ListUsedModules(path string) ([]meta.Package, error) {
	return m.plugin.ListUsedModules(path)
}

// List Modules With Deps ...
func (m *PIP) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	return m.plugin.ListModulesWithDeps(path, globalSettingFile)
}
