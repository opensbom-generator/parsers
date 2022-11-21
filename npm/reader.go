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
		} else {
			return nil
		}
	}

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
	lock := PackageLockV2
	// fill in Name, Version and LockfileVersion
	PackageLockV2.Name = data["name"]
	PackageLockV2.Version = data["version"]
	PackageLockV2.LockfileVersion = data["lockfileVersion"]
	PackageLockV2.Requires = data["requires"]
	// fill in Packages
	packages := data["packages"].(map[string]interface{})
	for pkg_name, pkg_obj := range packages {

	}
	// fill in Dependencies
	dependencies := data["dependencies"].(map[string]interface{})
	for dep_name, dep_obj := range dependencies {

	}
}
