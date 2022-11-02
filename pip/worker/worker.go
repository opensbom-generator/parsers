// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
)

// Virtual env constants
const (
	manifestSetupPy  = "setup.py"
	manifestSetupCfg = "setup.cfg"
	ModuleDotVenv    = ".venv"
	ModuleVenv       = "venv"
	PyvenvCfg        = "pyvenv.cfg"
	VirtualEnv       = "VIRTUAL_ENV"
)

var (
	errorWheelFileNotFound            = fmt.Errorf("wheel file not found")
	errorUnableToOpenWheelFile        = fmt.Errorf("unable to open wheel file")
	errorUnableToFetchPackageMetadata = fmt.Errorf("unable to fetch package details")
)

func IsRequirementMeet(data string) bool {
	_modules := LoadModules(data, "")
	return len(_modules) > 3
}

func GetVenFromEnvs() (bool, string, string) {
	venvfullpath := os.Getenv(VirtualEnv)
	splitstr := strings.Split(venvfullpath, "/")
	venv := splitstr[len(splitstr)-1]
	if len(venvfullpath) > 0 {
		return true, venv, venvfullpath
	}
	return false, venv, venvfullpath
}

func HasDefaultVenv(path string) (bool, string, string) {
	modules := []string{ModuleDotVenv, ModuleVenv}
	for i := range modules {
		venvfullpath := filepath.Join(path, modules[i])
		if helper.Exists(filepath.Join(path, modules[i])) {
			return true, modules[i], venvfullpath
		}
	}
	return false, "", ""
}

func HasPyvenvCfg(path string) bool {
	return helper.Exists(filepath.Join(path, PyvenvCfg))
}

func ScanPyvenvCfg(files *string, folderpath *string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if info.IsDir() {
			if HasPyvenvCfg(path) {
				*files = info.Name()
				p, _ := filepath.Abs(path)
				*folderpath = p

				// This is to break the walk for first enviironment found.
				// The assumption is there will be only one environment present
				return io.EOF
			}
		}
		return nil
	}
}

func SearchVenv(path string) (bool, string, string) {
	var (
		venv         string
		venvfullpath string
		state        bool
	)
	// if virtual env is active
	state, venv, venvfullpath = GetVenFromEnvs()
	if state {
		return true, venv, venvfullpath
	}

	state, venv, venvfullpath = HasDefaultVenv(path)
	if state {
		return state, venv, venvfullpath
	}

	err := filepath.Walk(path, ScanPyvenvCfg(&venv, &venvfullpath))
	if errors.Is(err, io.EOF) {
		err = nil
	}

	if err == nil {
		return true, venv, venvfullpath
	}

	return false, venv, venvfullpath
}

func IsValidRootModule(path string) bool {
	modules := []string{manifestSetupCfg, manifestSetupPy}
	for i := range modules {
		if helper.Exists(filepath.Join(path, modules[i])) {
			return true
		}
	}
	return false
}

func IsRootModule(pkg Packages, pipType string) bool {
	osWin := "windows"
	osDarwin := "darwin"
	osLinux := "linux"
	pipenv := "pipenv"
	poetry := "poetry"
	pyenv := "pyenv"
	os := runtime.GOOS
	switch {
	case os == osWin && (pipType == pipenv || pipType == pyenv):
		if !strings.Contains(pkg.Location, "\\src\\") && !strings.Contains(pkg.Location, "\\site-packages") {
			return true
		}
	case (os == osDarwin || os == osLinux) && (pipType == pipenv || pipType == pyenv):
		if !strings.Contains(pkg.Location, "/src/") && !strings.Contains(pkg.Location, "/site-packages") {
			return true
		}
	case (os == osWin || os == osLinux || os == osDarwin) && pipType == poetry:
		if pkg.Installer == poetry {
			return true
		}
	default:
	}

	return false
}
