// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022, Oracle and/or its affiliates.

package npm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/spdx/tools-golang/spdx/v2_1"
	"github.com/spdx/tools-golang/spdx/v2_2"
	"github.com/spdx/tools-golang/spdx/v2_3"
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

// MakeSpdxObject instantiates an SPDX object based on the provided SPDX version
func MakeSpdxDocVersion(spdx_version string) (interface{}, error) {
	switch spdx_version {
	case "2.1":
		return v2_1.Document, nil
	case "2.2":
		return v2_2.Document, nil
	case "2.3":
		return v2_3.Document, nil
	default:
		return nil, errors.New("Unknown or unsupported SPDX Document version")
	}
}

// ManifestV2ToSpdx converts manifest data conforming to npm lockfile version 2
// to an SPDX object of a given version.
// A generic interface is provided to support the different SPDX objects
func ManifestV2ToSpdx(spdx_version string, data map[string]interface{}) (interface{}, error) {
	spdx_doc, err := MakeSpdxDocVersion(spdx_version)
	if err != nil {
		return nil, err
	}
	//
}

// GetSpdxDoc detects a manifest file in the given project path.
// Depending on what the spdx_version is, the corresponding SPDX
// model is created, filled in, and returned.
func GetSpdxDoc(proj_path, spdx_version string) (interface{}, error) {
	// detect manifest existing in the path
	manifest_file := DetectManifest(proj_path)
	if manifest_file == nil {
		// no manifest file detected, so return an error
		return nil, errors.New("No npm manifest file detected")
	}
	// read the manifest file
	content, err := ReadManifest(manifest_file)
	if err != nil {
		return nil, err
	}

}
