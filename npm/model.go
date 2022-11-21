// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022, Oracle and/or its affiliates.

package npm

// PackageV2 represents the NPM V2 lockfile's package
// object. The same object is used for "Packages"
// and "Dependencies".
type PackageV2 struct {
	Name            string
	Version         string
	License         string
	Dependencies    map[string]string
	DevDependencies map[string]string
	Requires        map[string]string
	Engines         map[string]string
	Resolved        string `json:Resolved,omitempty`
	Integrity       string `json:Integrity,omitempty`
	Dev             bool   `json:Dev,omitempty`
}

// PackageLockV2 represents the NPM v2 lockfile
type PackageLockV2 struct {
	Name            string
	Version         string
	LockfileVersion int
	Requires        bool
	Packages        map[string]PackageV2
	Dependencies    map[string]PackageV2
}
