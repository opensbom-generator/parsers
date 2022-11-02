// SPDX-License-Identifier: Apache-2.0

package pipenv

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/pip/worker"
	"github.com/opensbom-generator/parsers/plugin"
)

const cmdName = "pipenv"
const manifestFile = "Pipfile"
const manifestLockFile = "Pipfile.lock"
const placeholderPkgName = "{PACKAGE}"
const packageSrcLocation = "/src/"
const packageSiteLocation = "/site-packages"

var errDependenciesNotFound = errors.New("unable to generate SPDX file: no modules or vendors found. Please install them before running spdx-sbom-generator, e.g.: `pipenv install` or `pipenv update`")
var errBuildlingModuleDependencies = errors.New("error building module dependencies")
var errNoPipCommand = errors.New("cannot find the pipenv command")
var errVersionNotFound = errors.New("python version not found")
var errFailedToConvertModules = errors.New("failed to convert modules")

type PipEnv struct {
	metadata   plugin.Metadata
	rootModule *meta.Package
	command    *helper.Cmd
	basepath   string
	version    string
	pkgs       []worker.Packages
	metainfo   map[string]worker.Metadata
	allModules []meta.Package
}

// New ...
func New() *PipEnv {
	return &PipEnv{
		metadata: plugin.Metadata{
			Name:       "The Python Package Index (PyPI)",
			Slug:       "pipenv",
			Manifest:   []string{manifestLockFile},
			ModulePath: []string{},
		},
	}
}

// Get Metadata ...
func (m *PipEnv) GetMetadata() plugin.Metadata {
	return m.metadata
}

// Is Valid ...
func (m *PipEnv) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if helper.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

// Has Modules Installed ...
func (m *PipEnv) HasModulesInstalled(path string) error {
	if err := m.buildCmd(ModulesCmd, m.basepath); err != nil {
		return err
	}
	result, err := m.command.Output()
	if err == nil && len(result) > 0 && worker.IsRequirementMeet(result) {
		return nil
	}
	return errDependenciesNotFound
}

// Get Version ...
func (m *PipEnv) GetVersion() (string, error) {
	if err := m.buildCmd(VersionCmd, m.basepath); err != nil {
		return "", err
	}
	version, err := m.command.Output()
	m.version = worker.GetShortPythonVersion(version)
	if err != nil {
		return "Python", errVersionNotFound
	}
	return version, err
}

// Set Root Module ...
func (m *PipEnv) SetRootModule(path string) error {
	m.basepath = path
	return nil
}

// Get Root Module ...
func (m *PipEnv) GetRootModule(path string) (*meta.Package, error) {
	if m.rootModule == nil {
		module := m.fetchRootModule()
		m.rootModule = &module
	}
	return m.rootModule, nil
}

// List Used Modules...
func (m *PipEnv) ListUsedModules(path string) ([]meta.Package, error) {
	if err := m.LoadModuleList(path); err != nil {
		return m.allModules, errFailedToConvertModules
	}

	decoder := worker.NewMetadataDecoder(m.GetPackageDetails)
	metainfo, err := decoder.ConvertMetadataToModules(m.pkgs, &m.allModules)
	if err != nil {
		return m.allModules, err
	}

	m.metainfo = metainfo
	return m.allModules, nil
}

// List Modules With Deps ...
func (m *PipEnv) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	modules, err := m.ListUsedModules(path)
	if err != nil {
		return nil, err
	}
	if _, err := m.GetRootModule(path); err != nil {
		return nil, err
	}
	if err := worker.BuildDependencyGraph(&m.allModules, &m.metainfo); err != nil {
		return nil, err
	}
	return modules, err
}

func (m *PipEnv) buildCmd(cmd command, path string) error {
	cmdArgs := cmd.Parse()
	if cmdArgs[0] != cmdName {
		return errNoPipCommand
	}

	command := helper.NewCmd(helper.CmdOptions{
		Name:      cmdArgs[0],
		Args:      cmdArgs[1:],
		Directory: path,
	})

	m.command = command

	return command.Build()
}

func (m *PipEnv) GetPackageDetails(packageNameList string) (string, error) {
	metatdataCmd := command(strings.ReplaceAll(string(MetadataCmd), placeholderPkgName, packageNameList))

	if err := m.buildCmd(metatdataCmd, m.basepath); err != nil {
		return "", err
	}
	result, err := m.command.Output()
	if err != nil {
		return "", err
	}

	return result, nil
}

func (m *PipEnv) PushRootModuleToVenv() (bool, error) {
	if err := m.buildCmd(InstallRootModuleCmd, m.basepath); err != nil {
		return false, err
	}
	result, err := m.command.Output()
	if err == nil && len(result) > 0 {
		return true, err
	}
	return false, nil
}

func (m *PipEnv) markRootModue() {
	for i, pkg := range m.pkgs {
		if worker.IsRootModule(pkg, m.metadata.Slug) {
			m.pkgs[i].Root = true
			break
		}
	}
}

func (m *PipEnv) LoadModuleList(path string) error {
	var state bool
	var err error

	if worker.IsValidRootModule(path) {
		state, err = m.PushRootModuleToVenv()
		if err != nil && !state {
			return err
		}
		if err := m.buildCmd(ModulesCmd, m.basepath); err != nil {
			return err
		}
		result, err := m.command.Output()
		if err == nil && len(result) > 0 && worker.IsRequirementMeet(result) {
			m.pkgs, err = worker.LoadModules(result, m.version)
			if err != nil {
				return fmt.Errorf("loading modules: %w", err)
			}
			m.markRootModue()
		}
		return err
	}
	return err
}

func (m *PipEnv) fetchRootModule() meta.Package {
	for _, mod := range m.allModules {
		if mod.Root {
			return mod
		}
	}
	return meta.Package{}
}
