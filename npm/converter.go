// SPDX-License-Identifier: Apache-2.0
// Copyright 2023 The Linux Foundation and its contributors

package npm

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/opensbom-generator/parsers/meta"
)

// ParseIntegrity takes the npm 'integrity' value and returns either a
// Checksum object or an error
func ParseIntegrityV2(i string) (*meta.Checksum, error) {
	// NPM's specification follows https://w3c.github.io/webappsec-subresource-integrity/
	// This is a rough implementation of the metadata parsing algorithm
	iSlice := strings.Split(i, "-")
	algoType := iSlice[0]
	// the hash value is base64 encoded, hence we have to decode it
	algoValByte, err := base64.StdEncoding.DecodeString(iSlice[1])
	cs := meta.Checksum{}
	if err != nil {
		// if this value is undecodable, we have a problem
		return nil, err
	}
	cs.Algorithm = meta.GetHashAlgorithm(algoType)
	cs.Content = algoValByte
	cs.Value = fmt.Sprintf("%x", algoValByte)
	return &cs, nil
}

// PackageV2ToMeta takes an npm v2 package name and PackageV2 object
// and returns a meta Package object.
func PackageV2ToMeta(name string, p PackageV2) (*meta.Package, error) {
	// Parse Integrity first
	cs, err := ParseIntegrityV2(p.Integrity)
	if err != nil {
		return nil, err
	}
	m := meta.Package{
		Version: p.Version,
		Name:    name,
		// Path
		// Local Path
		Supplier: meta.Supplier{}, //NPM lock files don't have a supplier
		// PackageURL
		Checksum: *cs,
		// PackageHomePage
		PackageDownloadLocation: p.Resolved,
		// LicenseConcluded
		// LicenseDeclared
		// CommentsLicense
		// OtherLicense
		// Copyright
		// PackageComment
		Root: false,
		// Packages
	}
	return &m, nil
}

// RootPackageV2ToMeta takes a RootPackageV2 object
// and returns a meta Package object.
func RootPackageV2ToMeta(p RootPackageV2) (*meta.Package, error) {
	m := meta.Package{
		Version: p.Version,
		Name:    p.Name,
		// Path
		// Local Path
		Supplier: meta.Supplier{}, //NPM lock files don't have a supplier
		// PackageURL
		// npm root package does not have a checksum
		// so we will calculate a SHA512
		Checksum: meta.Checksum{Algorithm: meta.HashAlgoSHA512},
		// PackageHomePage
		// PackageDownloadLocation
		// LicenseConcluded
		LicenseDeclared: p.License,
		// CommentsLicense
		// OtherLicense
		// Copyright
		// PackageComment
		Root: true,
		// Packages
	}
	return &m, nil
}
