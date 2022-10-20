// SPDX-License-Identifier: Apache-2.0

package models

import "github.com/opensbom-generator/parsers/meta"

// IPlugin ...
type IPlugin interface {
	SetRootModule(path string) error
	GetVersion() (string, error)
	GetMetadata() PluginMetadata
	GetRootModule(path string) (*meta.Package, error)
	ListUsedModules(path string) ([]meta.Package, error)
	ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error)
	IsValid(path string) bool
	HasModulesInstalled(path string) error
}

// PluginMetadata ...
type PluginMetadata struct {
	Name       string
	Slug       string
	Manifest   []string
	ModulePath []string
}

// OutputFormat defines an int enum of supported output formats
type OutputFormat int

const (
	OutputFormatSpdx OutputFormat = iota
	OutputFormatJson
)
