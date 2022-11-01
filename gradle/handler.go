// SPDX-License-Identifier: Apache-2.0

package javagradle

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
)

type gradle struct {
	metadata plugin.Metadata
	ge       gradleExec
	basepath string
}

func New() *gradle {
	return &gradle{
		metadata: plugin.Metadata{
			Name:       "Java Gradle",
			Slug:       "Java-Gradle",
			Manifest:   []string{"build.gradle", "settings.gradle"},
			ModulePath: []string{"."},
		},
	}
}

func (m *gradle) GetMetadata() plugin.Metadata {
	return m.metadata
}

func (m *gradle) SetRootModule(path string) error {
	m.basepath = path
	m.ge = newGradleExec(path)
	return nil
}

func (m *gradle) IsValid(path string) bool {
	for i := range m.metadata.Manifest {
		if helper.Exists(filepath.Join(path, m.metadata.Manifest[i])) {
			return true
		}
	}
	return false
}

func (m *gradle) GetVersion() (string, error) {
	cmd := m.ge.run("--version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (m *gradle) GetRootModule(path string) (*meta.Package, error) {
	// this doesn't actually do anything and is not called by any
	// orchestrator, should it still be in the interface?
	return nil, fmt.Errorf("GetRootModule not implemented for java-gradle")
}

func (m *gradle) ListUsedModules(path string) ([]meta.Package, error) {
	// this doesn't actually do anything and is not called by any
	// orchestrator, should it still be in the interface?
	return nil, fmt.Errorf("ListUsedModules not implemented for java-gradle")
}

func (m *gradle) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	pi, err := getProjectInfo(path)
	if err != nil {
		return nil, err
	}
	rootModule := meta.Package{
		Name:    pi.name,
		Version: pi.version,
		Supplier: meta.Supplier{
			Type: "Group Id",
			Name: pi.group,
		},
		Root:     true,
		Packages: make(map[string]*meta.Package),
	}
	// mediocre effort to read git info
	origin, sha1, err := getGitInfo(path)
	if err != nil {
		rootModule.Checksum = meta.Checksum{
			Algorithm: meta.HashAlgorithm("None"),
			Value:     "none",
		}
	} else {
		rootModule.Checksum = meta.Checksum{
			Algorithm: meta.HashAlgoSHA1,
			Value:     sha1,
		}
		rootModule.PackageDownloadLocation = origin
	}
	all, err := getDependencyModules(rootModule, path)
	if err != nil {
		return nil, err
	}
	return all, nil
}

func getDependencyModules(project meta.Package, path string) ([]meta.Package, error) {
	modsMap := map[string]*meta.Package{}
	mods := []meta.Package{project}

	deps, err := getDependencies(path)
	if err != nil {
		return nil, err
	}
	repos, err := getRepositories(path)
	if err != nil {
		return nil, err
	}
	depLoc, err := findDownloadLocations(repos, deps.all)
	if err != nil {
		return nil, err
	}

	for dep, remote := range depLoc {
		mod, err := generateModule(dep, remote)
		if err != nil {
			return nil, err
		}
		mods = append(mods, mod)
		modsMap[dep] = &mod
	}

	// add all root dependencies to the project module
	for _, rootDep := range deps.root {
		mod, ok := modsMap[rootDep]
		if !ok {
			return nil, fmt.Errorf("could not find module for %q", rootDep)
		}
		// apparently the key is just thrown away, so this just has to be something unique
		project.Packages[rootDep] = mod
	}

	// add transitive dependencies
	for dep, tdeps := range deps.graph {
		mod, ok := modsMap[dep]
		if !ok {
			return nil, fmt.Errorf("could not find module for %q", dep)
		}
		for _, tdep := range tdeps {
			tmod, ok := modsMap[tdep]
			if !ok {
				return nil, fmt.Errorf("could not find module for %q", tdep)
			}
			// apparently the key is just thrown away, so this just has to be something unique
			mod.Packages[tdep] = tmod
		}
	}
	return mods, nil
}

// generate gradle dependency module (non-root)
func generateModule(name, depURL string) (meta.Package, error) {
	mod := meta.Package{}
	groupID, artifactID, version, err := splitDep(name)
	if err != nil {
		return mod, err
	}
	sha1, err := getSHA1(depURL)
	if err != nil {
		return mod, err
	}
	mod.Supplier = meta.Supplier{
		Type: "Group Id",
		Name: groupID,
	}
	mod.Name = artifactID
	mod.Version = version
	mod.PackageDownloadLocation = depURL
	mod.Checksum = meta.Checksum{
		Algorithm: meta.HashAlgoSHA1,
		Value:     sha1,
	}
	mod.Packages = make(map[string]*meta.Package)
	mod.Root = false

	return mod, nil
}

func (m *gradle) HasModulesInstalled(path string) error {
	// check if root has gradlew wrapper script
	if hasGradlew(path) {
		return nil
	}

	// then check for gradle on system path
	fname, err := exec.LookPath("gradle")
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = filepath.Abs(fname)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
