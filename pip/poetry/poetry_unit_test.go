package poetry

import (
	"testing"

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
