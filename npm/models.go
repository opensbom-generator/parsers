// SPDX-License-Identifier: Apache-2.0
// Copyright 2023 The Linux Foundation and its contributors

package npm

// PackageV2 represents the NPM V2 lockfile's
// package object.
type PackageV2 struct {
	Version          string
	Resolved         string
	Integrity        string
	Dev              bool
	Engines          map[string]string
	Dependencies     map[string]string
	Bin              map[string]string
	Deprecated       string
	HasInstallScript bool
}

// RootPackageV2 represents the NPM V2 lockfile's
// root package object.
type RootPackageV2 struct {
	PackageV2
	Name            string
	License         string
	DevDependencies map[string]string
}

// PackageLockV2 represents the NPM v2 lockfile
type PackageLockV2 struct {
	Name            string
	Version         string
	LockfileVersion int
	Requires        bool
	RootPackage     RootPackageV2
	Packages        map[string]PackageV2
}
