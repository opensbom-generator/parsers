package poetry

import (
	"testing"

	"github.com/opensbom-generator/parsers/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDataPath = "testdata"

func setupPoetry(t *testing.T, path string) *Poetry {
	poetry := New()
	err := poetry.SetRootModule(path)
	require.NoError(t, err)

	return poetry
}

func TestSetRootModule(t *testing.T) {
	poetry := New()
	err := poetry.SetRootModule(testDataPath)
	assert.Nil(t, err)
	assert.Equal(t, "testdata", poetry.basepath)
}

func TestGetVersion(t *testing.T) {
	poetry := setupPoetry(t, testDataPath)
	version, err := poetry.GetVersion()
	assert.Nil(t, err)
	assert.Equal(t, "Python 3.11.4\n", version)
}

func TestGetMetadata(t *testing.T) {
	poetry := setupPoetry(t, testDataPath)
	metadata := poetry.GetMetadata()
	assert.Equal(t, "The Python Package Index (PyPI)", metadata.Name)
	assert.Equal(t, "poetry", metadata.Slug)
	assert.Equal(t, []string{"poetry.lock"}, metadata.Manifest)
	assert.Equal(t, []string{}, metadata.ModulePath)
}

func TestHasModulesInstalled(t *testing.T) {
	for _, tc := range map[string]struct {
		poetry     *Poetry
		path       string
		modulesLen int
		err        error
	}{
		"valid directory, should return nil err": {
			poetry:     setupPoetry(t, testDataPath),
			path:       testDataPath,
			modulesLen: 16,
			err:        nil,
		},
		"invalid directory, should return errDependenciesNotFound": {
			poetry:     setupPoetry(t, "invalid"),
			path:       "invalid",
			modulesLen: 0,
			err:        errDependenciesNotFound,
		},
	} {
		err := tc.poetry.HasModulesInstalled(tc.path)
		assert.ErrorIs(t, err, tc.err)
	}
}

func TestListUsedModules(t *testing.T) {
	for _, tc := range map[string]struct {
		poetry     *Poetry
		path       string
		modulesLen int
		err        error
	}{
		"valid directory, should return nil err and list of modules": {
			poetry:     setupPoetry(t, testDataPath),
			path:       testDataPath,
			modulesLen: 16,
			err:        nil,
		},
		"invalid directory, should return non-nil err": {
			poetry:     setupPoetry(t, "invalid"),
			path:       "invalid",
			modulesLen: 0,
			err:        errFailedToConvertModules,
		},
	} {
		modules, err := tc.poetry.ListUsedModules(tc.path)
		assert.ErrorIs(t, err, tc.err)
		assert.Equal(t, tc.modulesLen, len(modules))
	}
}

func TestGetPackageDetails(t *testing.T) {
	poetry := setupPoetry(t, testDataPath)

	for _, tc := range map[string]struct {
		packageName    string
		packageVersion string
		shouldErr      bool
	}{
		"should return package details for fastapi": {
			packageName:    "fastapi",
			packageVersion: "0.93.0",
			shouldErr:      false,
		},
		"invalid package, should return non-nil err": {
			packageName:    "invalid",
			packageVersion: "",
			shouldErr:      true,
		},
	} {
		packageDetails, err := poetry.GetPackageDetails(tc.packageName)
		if tc.shouldErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Contains(t, packageDetails, tc.packageName)
		}
		assert.Contains(t, packageDetails, tc.packageVersion)
	}
}

func TestGetRootModule(t *testing.T) {
	poetry := setupPoetry(t, testDataPath)

	// ListUsedModules needs to be called because it sets m.allModules
	_, err := poetry.ListUsedModules(testDataPath)
	require.NoError(t, err)

	metadata, err := poetry.GetRootModule(testDataPath)
	require.NoError(t, err)

	assert.True(t, metadata.Root)
	assert.Equal(t, "anyio", metadata.Name)
	assert.Equal(t, "3.6.2", metadata.Version)
	assert.Equal(t, "pypi.org/pypi/anyio", metadata.Path)
	assert.Equal(t, "pypi.org/pypi/anyio/3.6.2", metadata.PackageURL)
	assert.Equal(t, meta.Checksum{
		Algorithm: meta.HashAlgoSHA256,
		Content:   []byte{97, 110, 121, 105, 111},
		Value:     "fbbe32bd270d2a2ef3ed1c5d45041250284e31fc0a4df4a5a6071842051a51e3",
	}, metadata.Checksum)
	assert.Equal(t, "MIT", metadata.LicenseDeclared)
	assert.Equal(t, "MIT", metadata.LicenseConcluded)
}
