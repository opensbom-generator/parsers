// SPDX-License-Identifier: Apache-2.0

package cargo_test

import (
	"errors"
	"testing"

	"github.com/opensbom-generator/parsers/cargo"
	"github.com/opensbom-generator/parsers/cargo/cargofakes"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/stretchr/testify/require"
)

func setupCargoMeta(m *cargo.Mod) {
	fake := &cargofakes.FakeCargoImplementation{}
	fake.GetCargoMetadataIfNeededReturns(nil, errors.New("synthetic error"))
	m.SetImplementation(fake)
}

func TestListUsedModules(t *testing.T) {
	err := errors.New("synthetic error")
	for _, tc := range []struct {
		prepare   func(*cargo.Mod)
		shouldErr bool
	}{
		// GetCargoMetadataIfNeeded fails
		{
			setupCargoMeta,
			true,
		},
		// GetRootModule fails
		{
			func(m *cargo.Mod) {
				fake := &cargofakes.FakeCargoImplementation{}
				fake.GetCargoMetadataIfNeededReturns(nil, nil)
				fake.GetRootModuleReturns(meta.Package{}, err)
				m.SetImplementation(fake)
			},
			true,
		},
		// PopulateDependencies fails
		{
			func(m *cargo.Mod) {
				fake := &cargofakes.FakeCargoImplementation{}
				fake.GetCargoMetadataIfNeededReturns(nil, nil)
				fake.GetRootModuleReturns(meta.Package{}, nil)
				fake.PopulateDependenciesReturns(err)
				m.SetImplementation(fake)
			},
			true,
		},
	} {
		sut := &cargo.Mod{}
		tc.prepare(sut)
		res, err := sut.ListUsedModules(".")
		if tc.shouldErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.NotNil(t, res)
		}
	}
}

func TestListModulesWithDeps(t *testing.T) {
	err := errors.New("synthetic error")
	for _, tc := range []struct {
		prepare   func(*cargo.Mod)
		shouldErr bool
	}{
		// GetCargoMetadataIfNeeded fails
		{
			setupCargoMeta,
			true,
		},
		// GetRootModule fails
		{
			func(m *cargo.Mod) {
				fake := &cargofakes.FakeCargoImplementation{}
				fake.GetRootModuleReturns(meta.Package{}, err)
				m.SetImplementation(fake)
			},
			true,
		},
		// PopulateDependencies fails
		{
			func(m *cargo.Mod) {
				fake := &cargofakes.FakeCargoImplementation{}
				fake.PopulateDependenciesReturns(err)
				m.SetImplementation(fake)
			},
			true,
		},
		// PopulateDependencies fails
		{
			func(m *cargo.Mod) {
				fake := &cargofakes.FakeCargoImplementation{}
				fake.PopulateDependenciesReturns(err)
				m.SetImplementation(fake)
			},
			true,
		},
	} {
		sut := &cargo.Mod{}
		tc.prepare(sut)
		res, err := sut.ListModulesWithDeps(".", "")
		if tc.shouldErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.NotNil(t, res)
		}
	}
}

func TestGetRootModule(t *testing.T) {
	err := errors.New("synthetic error")
	for _, tc := range []struct {
		prepare   func(*cargo.Mod)
		shouldErr bool
	}{
		// GetCargoMetadataIfNeeded fails
		{
			setupCargoMeta,
			true,
		},
		// GetRootProjectName fails
		{
			func(m *cargo.Mod) {
				fake := &cargofakes.FakeCargoImplementation{}
				fake.GetRootProjectNameReturns("", err)
				m.SetImplementation(fake)
			},
			true,
		},
	} {
		sut := &cargo.Mod{}
		tc.prepare(sut)
		res, err := sut.GetRootModule(".")
		if tc.shouldErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.NotNil(t, res)
		}
	}
}
