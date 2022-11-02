// SPDX-License-Identifier: Apache-2.0

package javamaven

import (
	"crypto/sha1" // nolint:gosec
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

type Maven struct {
	metadata   plugin.Metadata
	rootModule *meta.Package
	command    *helper.Cmd
}

// New ...
func New() *Maven {
	return &Maven{
		metadata: plugin.Metadata{
			Name:     "Java Maven",
			Slug:     "Java-Maven",
			Manifest: []string{"pom.xml"},
			// TODO: instead of vendor folder what to mention for java project
			// Currently checking for mvn executable path in PATH variable
			ModulePath: []string{"."},
		},
	}
}

// GetMetadata ...
func (m *Maven) GetMetadata() plugin.Metadata {
	return m.metadata
}

// SetRootModule ...
func (m *Maven) SetRootModule(path string) error {
	module, err := m.getModule(path)
	if err != nil {
		return err
	}

	m.rootModule = &module

	return nil
}

// IsValid ...
func (m *Maven) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if helper.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

// HasModulesInstalled ...
func (m *Maven) HasModulesInstalled(path string) error {
	// TODO: How to verify is java project is build
	// Enforcing mvn path to be set in PATH variable
	fname, err := exec.LookPath("mvn")
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = filepath.Abs(fname)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// GetVersion...
func (m *Maven) GetVersion() (string, error) {
	err := m.buildCmd(VersionCmd, ".")
	if err != nil {
		return "", err
	}

	return m.command.Output()
}

// GetRootModule...
func (m *Maven) GetRootModule(path string) (*meta.Package, error) {
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
func (m *Maven) ListUsedModules(path string) ([]meta.Package, error) {
	modules, err := convertPOMReaderToModules(path, true)

	if err != nil {
		log.Println(err)
		return modules, err
	}

	return modules, nil
}

// ListModulesWithDeps ...
func (m *Maven) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	modules, err := m.ListUsedModules(path)
	if err != nil {
		return nil, err
	}

	tdList, err := getTransitiveDependencyList(path, globalSettingFile)
	if err != nil {
		return nil, fmt.Errorf("getting mvn transitive dependency tree: %w", err)
	}

	buildDependenciesGraph(modules, tdList)

	return modules, nil
}

func (m *Maven) getModule(path string) (meta.Package, error) {
	modules, err := convertPOMReaderToModules(path, false)

	if err != nil {
		log.Println(err)
		return meta.Package{}, err
	}

	if len(modules) == 0 {
		return meta.Package{}, errFailedToConvertModules
	}

	return modules[0], nil
}

func (m *Maven) buildCmd(cmd command, path string) error {
	cmdArgs := cmd.Parse()

	command := helper.NewCmd(helper.CmdOptions{
		Name:      cmdArgs[0],
		Args:      cmdArgs[1:],
		Directory: path,
	})

	m.command = command

	return command.Build()
}

func readCheckSum(content string) string {
	h := sha1.New() // nolint:gosec
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}
