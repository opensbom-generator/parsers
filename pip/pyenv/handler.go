// SPDX-License-Identifier: Apache-2.0

package pyenv

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/pip/worker"
	"github.com/opensbom-generator/parsers/plugin"
)

const cmdName = "python"
const osWin = "windows"
const osDarwin = "darwin"
const osLinux = "linux"
const winExecutable = "Scripts"
const lxExecutable = "bin"
const manifestFile = "requirements.txt"
const placeholderPkgName = "{PACKAGE}"
const placeholderExecutableName = "{executable}"

var errDependenciesNotFound = errors.New("unable to generate SPDX file: no modules or vendors found. Please install them before running spdx-sbom-generator, e.g.: `pip install -r requirements.txt`")
var errBuildlingModuleDependencies = errors.New("error building module dependencies")
var errNoPipCommand = errors.New("cannot find the python command")
var errVersionNotFound = errors.New("python version not found")
var errFailedToConvertModules = errors.New("failed to convert modules")

type PyEnv struct {
	metadata   plugin.Metadata
	rootModule *meta.Package
	command    *helper.Cmd
	basepath   string
	version    string
	pkgs       []worker.Packages
	metainfo   map[string]worker.Metadata
	allModules []meta.Package
	venv       string
}

// New ...
func New() *PyEnv {
	return &PyEnv{
		metadata: plugin.Metadata{
			Name:       "The Python Package Index (PyPI)",
			Slug:       "pyenv",
			Manifest:   []string{manifestFile},
			ModulePath: []string{},
		},
	}
}

// Get Metadata ...
func (m *PyEnv) GetMetadata() plugin.Metadata {
	return m.metadata
}

// Is Valid ...
func (m *PyEnv) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if helper.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

// HasModulesInstalled
func (m *PyEnv) HasModulesInstalled(path string) error {
	dir := m.GetExecutableDir()
	ModulesCmd := GetExecutableCommand(ModulesCmd)
	if err := m.buildCmd(ModulesCmd, dir); err != nil {
		return err
	}
	result, err := m.command.Output()
	if err == nil && len(result) > 0 && worker.IsRequirementMeet(result) {
		return nil
	}
	return errDependenciesNotFound
}

// Get Version ...
func (m *PyEnv) GetVersion() (string, error) {
	version := "Python"
	err := errVersionNotFound

	runme := m.fetchVenvPath()
	if runme {
		dir := m.GetExecutableDir()
		VersionCmd := GetExecutableCommand(VersionCmd)
		if err = m.buildCmd(VersionCmd, dir); err != nil {
			return "", err
		}
		version, err = m.command.Output()
		m.version = worker.GetShortPythonVersion(version)
		if err != nil {
			version = "Python"
			err = errVersionNotFound
		}
	}
	return version, err
}

// Set Root Module ...
func (m *PyEnv) SetRootModule(path string) error {
	m.basepath = path
	return nil
}

// Get Root Module ...
func (m *PyEnv) GetRootModule(path string) (*meta.Package, error) {
	if m.rootModule == nil {
		module := m.fetchRootModule()
		m.rootModule = &module
	}
	return m.rootModule, nil
}

// List Used Modules...
func (m *PyEnv) ListUsedModules(path string) ([]meta.Package, error) {
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
func (m *PyEnv) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
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

func (m *PyEnv) buildCmd(cmd command, path string) error {
	cmdArgs := cmd.Parse()
	if !strings.Contains(cmdArgs[0], cmdName) {
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

func (m *PyEnv) GetExecutableDir() string {
	if len(m.metadata.ModulePath[0]) > 0 {
		return m.metadata.ModulePath[0]
	}
	return m.basepath
}

func (m *PyEnv) GetPackageDetails(packageName string) (string, error) {
	MetadataCmd := GetExecutableCommand(MetadataCmd)
	MetadataCmd = command(strings.ReplaceAll(string(MetadataCmd), placeholderPkgName, packageName))
	dir := m.GetExecutableDir()

	if err := m.buildCmd(MetadataCmd, dir); err != nil {
		return "", err
	}
	result, err := m.command.Output()
	if err != nil {
		return "", err
	}

	return result, nil
}

func (m *PyEnv) PushRootModuleToVenv() (bool, error) {
	dir := m.GetExecutableDir()
	InstallRootModuleCmd := GetExecutableCommand(InstallRootModuleCmd)
	if err := m.buildCmd(InstallRootModuleCmd, dir); err != nil {
		return false, err
	}
	result, err := m.command.Output()
	if err == nil && len(result) > 0 {
		return true, err
	}
	return false, nil
}

func (m *PyEnv) markRootModue() {
	for i, pkg := range m.pkgs {
		if worker.IsRootModule(pkg, m.metadata.Slug) {
			m.pkgs[i].Root = true
			break
		}
	}
}

func (m *PyEnv) LoadModuleList(path string) error {
	var state bool
	var err error

	if worker.IsValidRootModule(path) {
		state, err = m.PushRootModuleToVenv()
		if err != nil && !state {
			return err
		}
		dir := m.GetExecutableDir()
		ModulesCmd := GetExecutableCommand(ModulesCmd)
		if err := m.buildCmd(ModulesCmd, dir); err != nil {
			return err
		}
		result, err := m.command.Output()
		if err == nil && len(result) > 0 && worker.IsRequirementMeet(result) {
			m.pkgs = worker.LoadModules(result, m.version)
			m.markRootModue()
		}
		return err
	}
	return err
}

func (m *PyEnv) fetchRootModule() meta.Package {
	for _, mod := range m.allModules {
		if mod.Root {
			return mod
		}
	}
	return meta.Package{}
}

func (m *PyEnv) fetchVenvPath() bool {
	state, venv, venvpath := worker.SearchVenv(m.basepath)
	if state && len(venv) > 0 {
		m.venv = venv
		m.metadata.ModulePath = append(m.metadata.ModulePath, venvpath)
		return true
	}
	return false
}

func GetExecutableCommand(cmd command) command {
	os := runtime.GOOS
	switch os {
	case osWin:
		return command(strings.ReplaceAll(string(cmd), placeholderExecutableName, winExecutable))
	default:
		return command(strings.ReplaceAll(string(cmd), placeholderExecutableName, lxExecutable))
	}
}
