// SPDX-License-Identifier: Apache-2.0

package cargo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadLockFile(t *testing.T) {
	sut := defaultImplementation{}
	lock, err := sut.readLockFile("test_data")
	require.NoError(t, err)
	require.NotNil(t, lock)
	require.Len(t, lock.Packages, 70)

	// Ensure we are getting the transient data
	for _, p := range lock.Packages {
		if p.Name == "hyper" {
			require.Equal(t, p.Checksum, "02c929dc5c39e335a03c405292728118860721b10190d98c2a0f0efd5baafbac")
			require.Equal(t, p.Source, "registry+https://github.com/rust-lang/crates.io-index")
			require.Equal(t, p.Version, "0.14.20")
			require.Len(t, p.Dependencies, 16)
		}
	}
}

func TestReadConfigFile(t *testing.T) {
	sut := defaultImplementation{}
	conf, err := sut.readConfig("test_data/Cargo.toml")
	require.NoError(t, err)
	require.NotNil(t, conf)
	require.Len(t, conf.Dependencies, 3)
	// Ensure the 3 direct dependencies are there
	for _, d := range conf.Dependencies {
		ok := (d.Name == "pretty_env_logger") || (d.Name == "tokio") || (d.Name == "hyper")
		require.Truef(t, ok, "Unknown dependency name: %s", true)
	}
}

func TestGetCargoMetadata(t *testing.T) {
	sut := defaultImplementation{}
	data, err := sut.getCargoMetadata("test_data")
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Len(t, data.Packages, 56)
	require.True(t, strings.HasSuffix(data.WorkspaceRoot, "cargo/test_data"))
}

func TestGetRootProjectName(t *testing.T) {
	sut := defaultImplementation{}
	name, err := sut.getRootProjectName("test_data")
	require.NoError(t, err)
	require.NotEmpty(t, name)
	require.Equal(t, "hello-server", name)
}

func TestConvertCargoPackageToModule(t *testing.T) {
	sut := defaultImplementation{}
	cargoPackage := &Package{
		Name:        "aho-corasick",
		Version:     "0.7.18",
		ID:          "aho-corasick 0.7.18 (registry+https://github.com/rust-lang/crates.io-index)",
		Description: "Fast multiple substring searching.",
		Source:      "registry+https://github.com/rust-lang/crates.io-index",
		// Dependencies: []PackageDependency `json:"dependencies"`
		ManifestPath: "/home/johndoe/.cargo/registry/src/github.com-1ecc6299db9ec823/aho-corasick-0.7.18/Cargo.toml",
		Authors:      []string{"John Doe <johndow@example.com>"},
		Repository:   "https://github.com/BurntSushi/aho-corasick",
		Homepage:     "https://github.com/BurntSushi/aho-corasick",
		License:      "Unlicense/MIT",
	}
	metaPackage := sut.convertCargoPackageToMetaPackage(cargoPackage)
	require.Equal(t, cargoPackage.Name, metaPackage.Name)
	require.Equal(t, cargoPackage.Version, metaPackage.Version)
	require.Equal(t, "John Doe", metaPackage.Supplier.Name)
	require.Equal(t, "johndow@example.com", metaPackage.Supplier.Email)
	require.Equal(t, "Unlicense/MIT", metaPackage.LicenseDeclared)
}
