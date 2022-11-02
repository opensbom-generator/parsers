// SPDX-License-Identifier: Apache-2.0

package composer

import (
	"path/filepath"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

type Composer struct {
	metadata plugin.Metadata
	command  *helper.Cmd
}

// New ...
func New() *Composer {
	return &Composer{
		metadata: plugin.Metadata{
			Name:       "composer Package Manager",
			Slug:       "composer",
			Manifest:   []string{LockFileName},
			ModulePath: []string{VendorFolderName},
		},
	}
}

// GetMetadata ...
func (m *Composer) GetMetadata() plugin.Metadata {
	return m.metadata
}

// IsValid ...
func (m *Composer) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if helper.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

// HasModulesInstalled ...
func (m *Composer) HasModulesInstalled(path string) error {
	for i := range m.metadata.ModulePath {
		if helper.Exists(filepath.Join(path, m.metadata.ModulePath[i])) {
			return nil
		}
	}
	return errDependenciesNotFound
}

// GetVersion ...
func (m *Composer) GetVersion() (string, error) {
	if err := m.buildCmd(VersionCmd, "."); err != nil {
		return "", err
	}

	return m.command.Output()
}

// SetRootModule ...
func (m *Composer) SetRootModule(path string) error {
	return nil
}

// GetRootModule ...
func (m *Composer) GetRootModule(path string) (*meta.Package, error) {
	return nil, nil
}

// ListModulesWithDeps ...
func (m *Composer) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	return m.ListUsedModules(path)
}

// ListUsedModules...
func (m *Composer) ListUsedModules(path string) ([]meta.Package, error) {
	modules, err := m.getModulesFromComposerLockFile(path)
	if err != nil {
		return nil, errFailedToReadComposerFile
	}

	treeList, err := m.getTreeListFromComposerShowTree(path)
	if err != nil {
		return nil, errFailedToShowComposerTree
	}

	for _, treeComponent := range treeList.Installed {
		addTreeComponentsToModule(treeComponent, modules)
	}

	return modules, nil
}
