package poetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testDataPath = "testdata"

func setupPoetry(path string) *Poetry {
	poetry := New()
	err := poetry.SetRootModule(path)
	if err != nil {
		panic(err)
	}

	return poetry
}

func TestSetRootModule(t *testing.T) {
	poetry := New()
	err := poetry.SetRootModule(testDataPath)
	assert.Nil(t, err)
	assert.Equal(t, "testdata", poetry.basepath)
}

func TestGetVersion(t *testing.T) {
	poetry := setupPoetry(testDataPath)
	version, err := poetry.GetVersion()
	assert.Nil(t, err)
	assert.Equal(t, "Python 3.11.4\n", version)
}

func TestGetMetadata(t *testing.T) {
	poetry := setupPoetry(testDataPath)
	metadata := poetry.GetMetadata()
	assert.Equal(t, "The Python Package Index (PyPI)", metadata.Name)
	assert.Equal(t, "poetry", metadata.Slug)
	assert.Equal(t, []string{"poetry.lock"}, metadata.Manifest)
	assert.Equal(t, []string{}, metadata.ModulePath)
}

func TestGetPackageDetails(t *testing.T) {
	for _, tc := range map[string]struct {
		poetry     *Poetry
		path       string
		modulesLen int
		err        error
	}{
		"valid directory, should return nil err and list of modules": {
			poetry:     setupPoetry(testDataPath),
			path:       testDataPath,
			modulesLen: 16,
			err:        nil,
		},
		"invalid directory, should return non-nil err": {
			poetry:     setupPoetry("invalid"),
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
