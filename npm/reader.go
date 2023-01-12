// SPDX-License-Identifier: Apache-2.0
// Copyright 2023 The Linux Foundation and its contributors

package npm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// List of known manifest files in order of priority
var manifests = []string{"package-lock.json",
	"node_modules/.package-lock.json",
	"package.json",
	"npm-shrinkwrap.json",
}

// detectManifest detects what kind of manifest exists in the
// provided path
// and returns the path to the manifest that exists
func DetectManifest(path string) string {
	for i := range manifests {
		fullPath := filepath.Join(path, manifests[i])
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}
	return ""
}

// ReadManifest will read a JSON file and unmarshall the data
// into a map. The map is a key-value pair where the value is
// of an unknown type.
func ReadManifest(manifestFile string) (map[string]interface{}, error) {
	var data map[string]interface{}
	content, err := os.ReadFile(manifestFile)
	if err != nil {
		return nil, errors.New("cannot read manifest file")
	}
	err = json.Unmarshal(content, &data)
	if err != nil {
		return nil, errors.New("cannot unmarshal JSON data")
	}
	return data, nil
}

// ParseManifestV2 converts a map[string]interface{} object into
// a struct representing the NPM v2 lockfile
func ParseManifestV2(data map[string]interface{}) (PackageLockV2, error) {
	// PackageV2 and PackageLockV2 come from model.go
	lock := PackageLockV2{}
	// fill in Name, Version, LockfileVersion and Requires
	lock.Name = data["name"].(string)
	lock.Version = data["version"].(string)
	lock.LockfileVersion = int(data["lockfileVersion"].(float64))
	lock.Requires = data["requires"].(bool)
	// fill in Packages
	// For V2 lockfile versions, there is no need to read
	// dependencies because those contain an identical
	// list formatted differently
	lock.Packages = make(map[string]PackageV2)
	packages := data["packages"].(map[string]interface{})
	for pkgName, pkgVal := range packages {
		pkgData := pkgVal.(map[string]interface{})
		// fill in root package info
		if pkgName == "" {
			root := RootPackageV2{}
			root.Name = pkgData["name"].(string)
			root.Version = pkgData["version"].(string)
			root.License = pkgData["license"].(string)

			// fill in dependencies and dev dependencies
			// if present
			if dependencies, ok := pkgData["dependencies"].(map[string]interface{}); ok {
				root.Dependencies = make(map[string]string)
				for depName, depVer := range dependencies {
					root.Dependencies[depName] = depVer.(string)
				}
			}
			if devDependencies, ok := pkgData["devDependencies"].(map[string]interface{}); ok {
				root.DevDependencies = make(map[string]string)
				for devDepName, devDepVer := range devDependencies {
					root.DevDependencies[devDepName] = devDepVer.(string)
				}
			}
			lock.RootPackage = root
		} else {
			pkg := PackageV2{}
			pkg.Version = pkgData["version"].(string)
			pkg.Resolved = pkgData["resolved"].(string)
			pkg.Integrity = pkgData["integrity"].(string)
			// these values optionally appear in package-lock.json
			if dev, ok := pkgData["dev"].(bool); ok {
				pkg.Dev = dev
			}
			if engines, ok := pkgData["engines"].(map[string]interface{}); ok {
				pkg.Engines = make(map[string]string)
				for engineName, engineVer := range engines {
					pkg.Engines[engineName] = engineVer.(string)
				}
			}
			if dependencies, ok := pkgData["dependencies"].(map[string]interface{}); ok {
				pkg.Dependencies = make(map[string]string)
				for depName, depVer := range dependencies {
					pkg.Dependencies[depName] = depVer.(string)
				}
			}
			if bins, ok := pkgData["bin"].(map[string]interface{}); ok {
				pkg.Bin = make(map[string]string)
				for binName, binPath := range bins {
					pkg.Bin[binName] = binPath.(string)
				}
			}
			if deprecated, ok := pkgData["deprecated"]; ok {
				pkg.Deprecated = deprecated.(string)
			}
			if hasInstallScript, ok := pkgData["hasInstallScript"]; ok {
				pkg.HasInstallScript = hasInstallScript.(bool)
			}
			lock.Packages[pkgName] = pkg
		}
	}
	return lock, nil
}
