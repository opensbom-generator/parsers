// SPDX-License-Identifier: Apache-2.0

package swift

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

type Swift struct {
	metadata plugin.Metadata
}

const (
	ManifestFile   string = "Package.swift"
	BuildDirectory string = ".build"
)

// New creates a new Swift package instance
func New() *Swift {
	return &Swift{
		metadata: plugin.Metadata{
			Name:       "Swift Package Manager",
			Slug:       "swift",
			Manifest:   []string{ManifestFile},
			ModulePath: []string{BuildDirectory},
		},
	}
}

// GetVersion returns Swift language version
func (m *Swift) GetVersion() (string, error) {
	cmd := exec.Command("swift", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	version := string(output)

	return version, nil
}

// GetMetadata returns root package information base on path given
func (m *Swift) GetMetadata() plugin.Metadata {
	return m.metadata
}

// SetRootModule sets root package information base on path given
func (m *Swift) SetRootModule(path string) error {
	return nil
}

// GetRootModule returns root package information base on path given
func (m *Swift) GetRootModule(path string) (*meta.Package, error) {
	cmd := exec.Command("swift", "package", "describe", "--type", "json")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var description PackageDescription
	if err := json.NewDecoder(bytes.NewReader(output)).Decode(&description); err != nil {
		return nil, err
	}

	mod := description.Module()

	return mod, nil
}

// ListUsedModules fetches and lists
// all packages required by the project
// in the given project directory,
// this is a plain list of all used modules
// (no nested or tree view)
func (m *Swift) ListUsedModules(path string) ([]meta.Package, error) {
	cmd := exec.Command("swift", "package", "show-dependencies", "--disable-automatic-resolution", "--format", "json")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var root PackageDependency
	if err := json.NewDecoder(bytes.NewReader(output)).Decode(&root); err != nil {
		return nil, err
	}

	var dependencies []PackageDependency

	var recurse func(PackageDependency)
	recurse = func(dep PackageDependency) {
		for _, nested := range dep.Dependencies {
			dependencies = append(dependencies, nested)
			recurse(nested)
		}
	}
	recurse(root)

	collection := make([]meta.Package, len(dependencies))
	for _, dep := range dependencies {
		mod := dep.Module()
		collection = append(collection, *mod)
	}

	return collection, nil
}

// ListModulesWithDeps fetches and lists all packages
// (root and direct dependencies)
// required by the project in the given project directory (side-by-side),
// this is a one level only list of all used modules,
// and each with its direct dependency only
// (similar output to ListUsedModules but with direct dependency only)
func (m *Swift) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	var collection []meta.Package //nolint: prealloc

	mod, err := m.GetRootModule(path)
	if err != nil {
		return nil, err
	}
	collection = append(collection, *mod)

	cmd := exec.Command("swift", "package", "show-dependencies", "--disable-automatic-resolution", "--format", "json")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var root PackageDependency
	if err := json.NewDecoder(bytes.NewReader(output)).Decode(&root); err != nil {
		return nil, err
	}

	for _, dep := range root.Dependencies {
		mod := dep.Module()
		collection = append(collection, *mod)
	}

	return collection, nil
}

// IsValid checks if the project dependency file provided in the contract exists
func (m *Swift) IsValid(path string) bool {
	return helper.Exists(filepath.Join(path, ManifestFile))
}

// HasModulesInstalled checks whether
// the current project (based on given path)
// has the dependent packages installed
func (m *Swift) HasModulesInstalled(path string) error {
	if helper.Exists(filepath.Join(path, BuildDirectory)) {
		return nil
	}

	return errDependenciesNotFound
}
