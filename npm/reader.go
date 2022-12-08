// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022, Oracle and/or its affiliates.

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
		full_path := filepath.Join(path, manifests[i])
		if _, err := os.Stat(full_path); err == nil {
			return full_path
		}
	}
	return ""
}

// ReadManifest will read a JSON file and unmarshall the data
// into a map. The map is a key-value pair where the value is
// of an unknown type.
func ReadManifest(manifest_file string) (map[string]interface{}, error) {
	var data map[string]interface{}
	content, err := os.ReadFile(manifest_file)
	if err != nil {
		return nil, errors.New("Cannot read manifest file")
	}
	err = json.Unmarshal(content, &data)
	if err != nil {
		return nil, errors.New("Cannot unmarshal JSON data")
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
	for pkg_name, pkg_val := range packages {
		pkg_data := pkg_val.(map[string]interface{})
		// fill in root package info
		if pkg_name == "" {
			root := RootPackageV2{}
			root.Name = pkg_data["name"].(string)
			root.Version = pkg_data["version"].(string)
			root.License = pkg_data["license"].(string)

			// fill in dependencies and dev dependencies
			// if present
			if dependencies, ok := pkg_data["dependencies"].(map[string]interface{}); ok {
				root.Dependencies = make(map[string]string)
				for dep_name, dep_ver := range dependencies {
					root.Dependencies[dep_name] = dep_ver.(string)
				}
			}
			if devdependencies, ok := pkg_data["devDependencies"].(map[string]interface{}); ok {
				root.DevDependencies = make(map[string]string)
				for devdep_name, devdep_ver := range devdependencies {
					root.DevDependencies[devdep_name] = devdep_ver.(string)
				}
			}
			lock.RootPackage = root

		} else {
			pkg := PackageV2{}
			pkg.Version = pkg_data["version"].(string)
			pkg.Resolved = pkg_data["resolved"].(string)
			pkg.Integrity = pkg_data["integrity"].(string)
			// these values optionally appear in package-lock.json
			if dev, ok := pkg_data["dev"].(bool); ok {
				pkg.Dev = dev
			}
			if engines, ok := pkg_data["engines"].(map[string]interface{}); ok {
				pkg.Engines = make(map[string]string)
				for engine_name, engine_ver := range engines {
					pkg.Engines[engine_name] = engine_ver.(string)
				}
			}
			if dependencies, ok := pkg_data["dependencies"].(map[string]interface{}); ok {
				pkg.Dependencies = make(map[string]string)
				for dep_name, dep_ver := range dependencies {
					pkg.Dependencies[dep_name] = dep_ver.(string)
				}
			}
			if bins, ok := pkg_data["bin"].(map[string]interface{}); ok {
				pkg.Bin = make(map[string]string)
				for bin_name, bin_path := range bins {
					pkg.Bin[bin_name] = bin_path.(string)
				}
			}
			if deprecated, ok := pkg_data["deprecated"]; ok {
				pkg.Deprecated = deprecated.(string)
			}
			if hasInstallScript, ok := pkg_data["hasInstallScript"]; ok {
				pkg.HasInstallScript = hasInstallScript.(bool)
			}
			lock.Packages[pkg_name] = pkg

		}
	}
	return lock, nil
}
