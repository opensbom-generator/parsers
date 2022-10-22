// SPDX-License-Identifier: Apache-2.0

package modules

import (
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/opensbom-generator/parsers/cargo"
	"github.com/opensbom-generator/parsers/composer"
	"github.com/opensbom-generator/parsers/gem"
	gomod "github.com/opensbom-generator/parsers/go"
	javagradle "github.com/opensbom-generator/parsers/gradle"
	javamaven "github.com/opensbom-generator/parsers/maven"
	"github.com/opensbom-generator/parsers/npm"
	"github.com/opensbom-generator/parsers/nuget"
	"github.com/opensbom-generator/parsers/pip"
	"github.com/opensbom-generator/parsers/swift"
	"github.com/opensbom-generator/parsers/yarn"

	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

var (
	errNoPluginAvailable   = errors.New("no plugin system available for current path")
	errNoModulesInstalled  = errors.New("there are no components in the BOM. The project may not contain dependencies, please install modules")
	errFailedToReadModules = errors.New("failed to read modules")
)

var registeredPlugins []plugin.Plugin

func init() {
	registeredPlugins = append(registeredPlugins,
		cargo.New(),
		composer.New(),
		gomod.New(),
		gem.New(),
		npm.New(),
		javagradle.New(),
		javamaven.New(),
		nuget.New(),
		yarn.New(),
		pip.New(),
		swift.New(),
	)
}

// Manager ...
type Manager struct {
	Config  Config
	Plugin  plugin.Plugin
	modules []meta.Package
}

// Config ...
type Config struct {
	Path              string
	GlobalSettingFile string
}

// New ...
func New(cfg Config) ([]*Manager, error) {
	var usePlugin plugin.Plugin
	var managerSlice []*Manager
	for _, plugin := range registeredPlugins {
		if plugin.IsValid(cfg.Path) {
			if err := plugin.SetRootModule(cfg.Path); err != nil {
				return nil, err
			}

			usePlugin = plugin
			if usePlugin == nil {
				return nil, errNoPluginAvailable
			}

			managerSlice = append(managerSlice, &Manager{
				Config: cfg,
				Plugin: usePlugin,
			})
		}
	}

	return managerSlice, nil
}

// Run ...
func (m *Manager) Run() error {
	modulePath := m.Config.Path
	globalSettingFile := m.Config.GlobalSettingFile
	version, err := m.Plugin.GetVersion()
	if err != nil {
		return err
	}

	log.Infof("Current Language Version %s", version)
	log.Infof("Global Setting File %s", globalSettingFile)
	if moduleErr := m.Plugin.HasModulesInstalled(modulePath); moduleErr != nil {
		return moduleErr
	}

	modules, err := m.Plugin.ListModulesWithDeps(modulePath, globalSettingFile)
	if err != nil {
		log.Error(err)
		return errFailedToReadModules
	}

	m.modules = modules

	return nil
}

// GetSource ...
func (m *Manager) GetSource() []meta.Package {
	return m.modules
}
