// SPDX-License-Identifier: Apache-2.0

package gem

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

type Gem struct {
	metadata   plugin.Metadata
	rootModule *meta.Package
}

var errDependenciesNotFound, errInvalidProjectType = errors.New(
	`* Please install dependencies by running the following command :
    1) bundle config set --local path 'vendor/bundle' && bundle install && bundle exec rake install
    2) run the spdx-sbom-generator tool command`),
	errors.New(
		`* Tool only supports ruby gems projects with valid .gemspec manifest in project root directory`)

// New ...
func New() *Gem {
	return &Gem{
		metadata: plugin.Metadata{
			Name:       "Bundler",
			Slug:       "bundler",
			Manifest:   []string{"Gemfile", "Gemfile.lock", "gems.rb", "gems.locked"},
			ModulePath: []string{"vendor/bundle"},
		},
	}
}

// GetMetadata ...
func (g *Gem) GetMetadata() plugin.Metadata {
	return g.metadata
}

// IsValid ...
func (g *Gem) IsValid(path string) bool {

	for i := range g.metadata.Manifest {
		if helper.Exists(filepath.Join(path, g.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

// HasModulesInstalled ...
func (g *Gem) HasModulesInstalled(path string) error {

	if !validateProjectType(path) {
		return errInvalidProjectType
	}

	hasRake, _, hasModule := hasRakefile(path), ensurePlatform(path), false
	for i := range g.metadata.ModulePath {
		if helper.Exists(filepath.Join(path, g.metadata.ModulePath[i])) {
			hasModule = true
		}
	}
	if hasRake && hasModule {
		return nil
	}
	return errDependenciesNotFound
}

// GetVersion ...
func (g *Gem) GetVersion() (string, error) {

	cmd := exec.Command("bundler", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	fields := strings.Fields(string(output))

	if len(fields) > 1 && (fields[0] != "Bundler" || fields[1] != "version") {
		return "", fmt.Errorf("unexpected output format: %s", output)
	}
	if len(fields) < 2 {
		return "", fmt.Errorf("unexpected output format: %s", output)
	}
	return fields[2], nil
}

// SetRootModule ...
func (g *Gem) SetRootModule(path string) error {
	module, err := g.GetRootModule(path)
	if err != nil {
		return err
	}

	g.rootModule = module

	return nil
}

// GetRootModule...
func (g *Gem) GetRootModule(path string) (*meta.Package, error) {
	if err := g.HasModulesInstalled(path); err != nil {
		return &meta.Package{}, err
	}
	return getGemRootModule(path)
}

// GetModule ...
func (g *Gem) GetModule(path string) ([]meta.Package, error) {
	return nil, nil
}

// ListUsedModules ...
func (g *Gem) ListUsedModules(path string) ([]meta.Package, error) {
	var globalSettingFile string
	return g.ListModulesWithDeps(path, globalSettingFile)
}

// ListModulesWithDeps ...
func (g *Gem) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	if err := g.HasModulesInstalled(path); err != nil {
		return []meta.Package{}, err
	}
	return listGemRootModule(path)
}
