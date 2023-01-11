// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 The Linux Foundation and its contributors

package cargo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRootModule(t *testing.T) {
	mod := New()
	pkg, err := mod.GetRootModule("./testdata/")
	require.NoError(t, err)
	require.NotNil(t, pkg)
	require.Equal(t, "hello-server", pkg.Name)
	require.Equal(t, "Apache-2.0", pkg.LicenseDeclared)
	// TODO: Add more fields as they become available
}

func TestGetMetadata(t *testing.T) {
	mod := New()

	err := mod.SetRootModule("./testdata/")
	metadata := mod.GetMetadata()
	require.NoError(t, err)
	require.NotNil(t, metadata)
	require.Equal(t, "cargo", metadata.Slug)
	require.Equal(t, "Cargo.toml", metadata.Manifest[0])
}
func TestListUsedModules(t *testing.T) {
	mod := New()
	mods, err := mod.ListUsedModules("./testdata/")
	require.NoError(t, err)
	require.Len(t, mods, 3, "No jala salieron %s", len(mods))
}

func TestListModulesWithDeps(t *testing.T) {
	mod := New()
	mods, err := mod.ListModulesWithDeps("./testdata/", "")
	require.NoError(t, err)
	require.Len(t, mods, 3, len(mods))
	for _, p := range mods {
		switch p.Name {
		case "hyper":
			require.Lenf(t, p.Packages, 20, "%s has %d deps", p.Name, len(p.Packages))
		case "pretty_env_logger":
			require.Lenf(t, p.Packages, 2, "%s has %d deps", p.Name, len(p.Packages))
		case "tokio":
			require.Lenf(t, p.Packages, 12, "%s has %d deps", p.Name, len(p.Packages))
		default:
			t.Error("Unexpected name: " + p.Name)
		}
	}
	require.Len(t, mods, 3)
}
