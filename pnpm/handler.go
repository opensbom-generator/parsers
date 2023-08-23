// SPDX-License-Identifier: Apache-2.0

package pnpm

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	"github.com/opensbom-generator/parsers/plugin"
	"github.com/opensbom-generator/parsers/reader"
)

type Pnpm struct {
	metadata plugin.Metadata
}

var (
	errDependenciesNotFound = errors.New("unable to generate SPDX file, no modules founded. Please install them before running spdx-sbom-generator, e.g.: `pnpm install`")
	lockFile                = "pnpm-lock.yaml"
	rg                      = regexp.MustCompile(`^(((git|hg|svn|bzr)\+)?(http:\/\/www\.|https:\/\/www\.|http:\/\/|https:\/\/|ssh:\/\/|git:\/\/|svn:\/\/|sftp:\/\/|ftp:\/\/)?[a-z0-9]+([\-\.]{1}[a-z0-9]+){0,100}\.[a-z]{2,5}(:[0-9]{1,5})?(\/.*))|(git\+git@[a-zA-Z0-9\.]+:[a-zA-Z0-9/\\.@]+)|(bzr\+lp:[a-zA-Z0-9\.]+)$`)
)

// New creates a new pnpm instance
func New() *Pnpm {
	return &Pnpm{
		metadata: plugin.Metadata{
			Name:       "Performant Node Package Manager",
			Slug:       "pnpm",
			Manifest:   []string{"package.json", lockFile},
			ModulePath: []string{"node_modules"},
		},
	}
}

// GetMetadata returns metadata descriptions Name, Slug, Manifest, ModulePath
func (m *Pnpm) GetMetadata() plugin.Metadata {
	return m.metadata
}

// IsValid checks if module has a valid Manifest file
// for pnpm manifest file is package.json
func (m *Pnpm) IsValid(path string) bool {
	for _, p := range m.metadata.Manifest {
		if !helper.Exists(filepath.Join(path, p)) {
			return false
		}
	}
	return true
}

// HasModulesInstalled checks if modules of manifest file already installed
func (m *Pnpm) HasModulesInstalled(path string) error {
	for _, p := range m.metadata.ModulePath {
		if !helper.Exists(filepath.Join(path, p)) {
			return errDependenciesNotFound
		}
	}

	for _, p := range m.metadata.Manifest {
		if !helper.Exists(filepath.Join(path, p)) {
			return errDependenciesNotFound
		}
	}
	return nil
}

// GetVersion returns pnpm version
func (m *Pnpm) GetVersion() (string, error) {
	cmd := exec.Command("pnpm", "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	if len(strings.Split(string(output), ".")) != 3 {
		return "", fmt.Errorf("unexpected version format: %s", output)
	}

	return string(output), nil
}

// SetRootModule ...
func (m *Pnpm) SetRootModule(path string) error {
	return nil
}

// GetRootModule return
// root package information ex. Name, Version
func (m *Pnpm) GetRootModule(path string) (*meta.Package, error) {
	r := reader.New(filepath.Join(path, m.metadata.Manifest[0]))
	pkResult, err := r.ReadJSON()
	if err != nil {
		return &meta.Package{}, err
	}
	mod := &meta.Package{}
	mod.Root = true

	if pkResult["name"] != nil {
		mod.Name = pkResult["name"].(string)
	}
	if pkResult["author"] != nil {
		mod.Supplier.Name = pkResult["author"].(string)
	}
	if pkResult["version"] != nil {
		mod.Version = pkResult["version"].(string)
	}
	repository := pkResult["repository"]
	if repository != nil {
		if rep, ok := repository.(string); ok {
			mod.PackageDownloadLocation = rep
		}
		if _, ok := repository.(map[string]interface{}); ok && repository.(map[string]interface{})["url"] != nil {
			mod.PackageDownloadLocation = repository.(map[string]interface{})["url"].(string)
		}
	}
	if pkResult["homepage"] != nil {
		mod.PackageURL = helper.RemoveURLProtocol(pkResult["homepage"].(string))
		mod.PackageDownloadLocation = mod.PackageURL
	}
	if !rg.MatchString(mod.PackageDownloadLocation) {
		mod.PackageDownloadLocation = "NONE"
	}
	mod.Packages = map[string]*meta.Package{}
	mod.Copyright = getCopyright(path)
	modLic, err := helper.GetLicenses(path)
	if err != nil {
		return mod, nil
	}
	mod.LicenseDeclared = helper.BuildLicenseDeclared(modLic.ID)
	mod.LicenseConcluded = helper.BuildLicenseConcluded(modLic.ID)
	mod.CommentsLicense = modLic.Comments
	if !helper.LicenseSPDXExists(modLic.ID) {
		mod.OtherLicense = append(mod.OtherLicense, *modLic)
	}
	return mod, nil
}

// ListUsedModules return brief info of installed modules, Name and Version
func (m *Pnpm) ListUsedModules(path string) ([]meta.Package, error) {
	r := reader.New(filepath.Join(path, m.metadata.Manifest[0]))
	pkResult, err := r.ReadJSON()
	if err != nil {
		return []meta.Package{}, err
	}
	packages := make([]meta.Package, 0)
	deps := pkResult["dependencies"].(map[string]interface{})

	for k, v := range deps {
		var mod meta.Package
		mod.Name = k
		mod.Version = strings.TrimPrefix(v.(string), "^")
		packages = append(packages, mod)
	}

	return packages, nil
}

// ListModulesWithDeps return all info of installed modules
func (m *Pnpm) ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error) {
	deps, err := readLockFile(filepath.Join(path, lockFile))
	if err != nil {
		return nil, err
	}
	allDeps := appendNestedDependencies(deps)
	return m.buildDependencies(path, allDeps)
}

func (m *Pnpm) buildDependencies(path string, deps []dependency) ([]meta.Package, error) {
	modules := make([]meta.Package, 0)
	de, err := m.GetRootModule(path)
	if err != nil {
		return modules, err
	}
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s-%s", de.Name, de.Version))))
	de.Checksum = meta.Checksum{
		Algorithm: "SHA256",
		Value:     h,
	}
	de.Supplier.Name = de.Name
	if de.PackageDownloadLocation == "" {
		de.PackageDownloadLocation = de.Name
	}
	modules = append(modules, *de)
	for _, d := range deps {
		var mod meta.Package
		mod.Name = d.Name
		mod.Version = extractVersion(d.Version)
		modules[0].Packages[d.Name] = &meta.Package{
			Name:     d.Name,
			Version:  mod.Version,
			Checksum: meta.Checksum{Content: []byte(fmt.Sprintf("%s-%s", d.Name, mod.Version))},
		}
		if len(d.Dependencies) != 0 {
			mod.Packages = map[string]*meta.Package{}
			for _, depD := range d.Dependencies {
				ar := strings.Split(strings.TrimSpace(depD), " ")
				name := strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(ar[0], "\""), "\""), "@")
				if name == "optionalDependencies:" {
					continue
				}

				version := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(ar[1]), "\""), "\"")
				if extractVersion(version) == "*" {
					continue
				}
				mod.Packages[name] = &meta.Package{
					Name:     name,
					Version:  extractVersion(version),
					Checksum: meta.Checksum{Content: []byte(fmt.Sprintf("%s-%s", name, version))},
				}
			}
		}
		mod.PackageDownloadLocation = strings.TrimSuffix(strings.TrimPrefix(d.Resolved, "\""), "\"")
		mod.Supplier.Name = mod.Name

		mod.PackageURL = getPackageHomepage(filepath.Join(path, m.metadata.ModulePath[0], d.PkPath, m.metadata.Manifest[0]))
		h := fmt.Sprintf("%x", sha256.Sum256([]byte(mod.Name)))
		mod.Checksum = meta.Checksum{
			Algorithm: "SHA256",
			Value:     h,
		}

		licensePath := filepath.Join(path, m.metadata.ModulePath[0], d.PkPath, "LICENSE")

		libDirName := fmt.Sprintf("%s@%s", strings.ReplaceAll(d.PkPath, "/", "+"), d.Version)
		if d.Belonging != "" {
			libDirName += fmt.Sprintf("%s_%s", libDirName, d.Belonging)
		}
		licensePathInsidePnpm := filepath.Join(
			path,
			m.metadata.ModulePath[0],
			".pnpm",
			libDirName,
			m.metadata.ModulePath[0],
			d.PkPath,
			"LICENSE",
		)

		var validLicensePath string
		switch {
		case helper.Exists(licensePath):
			validLicensePath = licensePath
		case helper.Exists(licensePathInsidePnpm):
			validLicensePath = licensePathInsidePnpm
		default:
			validLicensePath = ""
		}

		r := reader.New(validLicensePath)
		s := r.StringFromFile()
		mod.Copyright = helper.GetCopyright(s)

		modLic, err := helper.GetLicenses(filepath.Join(path, m.metadata.ModulePath[0], d.PkPath))
		if err != nil {
			modules = append(modules, mod)
			continue
		}
		mod.LicenseDeclared = helper.BuildLicenseDeclared(modLic.ID)
		mod.LicenseConcluded = helper.BuildLicenseConcluded(modLic.ID)
		mod.CommentsLicense = modLic.Comments
		if !helper.LicenseSPDXExists(modLic.ID) {
			mod.OtherLicense = append(mod.OtherLicense, *modLic)
		}
		modules = append(modules, mod)
	}
	return modules, nil
}

func readLockFile(path string) ([]dependency, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lockData map[string]interface{}
	err = yaml.Unmarshal(fileContent, &lockData)
	if err != nil {
		return nil, err
	}
	return readLockData(lockData)
}

func readLockData(lockData map[string]interface{}) ([]dependency, error) {
	packages, ok := lockData["packages"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid lock file format")
	}

	dependencies := make([]dependency, 0)

	lockfileVersion, ok := lockData["lockfileVersion"]
	if !ok {
		return nil, errors.New("no lockfileVersion field")
	}
	var lockfileVersionNum float64
	var err error
	// lockfileVersion can be float64 or string ...
	// 6.0-, it is float64
	// 6.0+, it is string ...
	switch lockfileVersion := lockfileVersion.(type) {
	case string:
		lockfileVersionNum, err = strconv.ParseFloat(lockfileVersion, 64)
		if err != nil {
			return nil, err
		}
	case float64:
		lockfileVersionNum = lockfileVersion
	default:
		return nil, fmt.Errorf("invalid lockfile version type")
	}

	for pkgName, pkg := range packages {
		pkgMap, ok := pkg.(map[string]interface{})
		if !ok {
			continue
		}

		dep := dependency{}

		var name, version, belonging string
		if lockfileVersionNum > 6.0 {
			name, version, belonging = splitPackageNameAndVersionV6(pkgName)
		} else {
			name, version, belonging = splitPackageNameAndVersionLegacy(pkgName)
		}

		nameWithoutAt, pkPath, nameAndVersion := processName(name)
		dep.Name = nameWithoutAt
		dep.PkPath = pkPath
		dep.Version = version
		dep.Belonging = belonging
		if resolution, ok := pkgMap["resolution"].(map[string]interface{}); ok {
			if tarball, ok := resolution["tarball"].(string); ok {
				dep.Resolved = tarball
			}
			if integrity, ok := resolution["integrity"].(string); ok {
				dep.Integrity = integrity
			}
		}
		if dep.Resolved == "" {
			// .npmrc
			registry := "https://registry.npmjs.org"
			dep.Resolved = fmt.Sprintf("%s/%s/-/%s-%s.tgz", registry, name, nameAndVersion, dep.Version)
		}

		dependenciesRaw, ok := pkgMap["dependencies"].(map[string]interface{})
		if ok {
			for depName, ver := range dependenciesRaw {
				depPath := fmt.Sprintf("%s %s", depName, ver)
				dep.Dependencies = append(dep.Dependencies, strings.TrimSpace(depPath))
			}
		}

		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

func getCopyright(path string) string {
	licensePath := filepath.Join(path, "LICENSE")
	if helper.Exists(licensePath) {
		r := reader.New(licensePath)
		s := r.StringFromFile()
		return helper.GetCopyright(s)
	}

	licenseMDPath, err := filepath.Glob(filepath.Join(path, "LICENSE*"))
	if err != nil {
		return ""
	}
	if len(licenseMDPath) > 0 && helper.Exists(licenseMDPath[0]) {
		r := reader.New(licenseMDPath[0])
		s := r.StringFromFile()
		return helper.GetCopyright(s)
	}

	return ""
}

func getPackageHomepage(path string) string {
	r := reader.New(path)
	pkResult, err := r.ReadJSON()
	if err != nil {
		return ""
	}
	if pkResult["homepage"] != nil {
		return helper.RemoveURLProtocol(pkResult["homepage"].(string))
	}
	return ""
}

func extractVersion(s string) string {
	t := strings.TrimPrefix(s, "^")
	t = strings.TrimPrefix(t, "~")
	t = strings.TrimPrefix(t, ">")
	t = strings.TrimPrefix(t, "=")

	t = strings.Split(t, " ")[0]
	return t
}

func splitPackageNameAndVersionLegacy(pkg string) (string, string, string) {
	// sample input (lockfile 6.0-
	// 1. /@byted-cmf/data-plugin-indexeddb-storage-client/2.0.4_e239e53d72e8372ca29c63c7108bdc0f
	// 2. /esprima/1.2.5
	// 3. /@dp/sirius-view/3.7.131
	// 4. /@babel/plugin-syntax-json-strings/7.8.3_@babel+core@7.15.0

	parts := strings.Split(pkg, "_")
	belonging := ""
	if len(parts) > 1 {
		belonging = strings.Join(parts[1:], "_")
	}
	pkgPure := strings.TrimSuffix(pkg, fmt.Sprintf("_%s", belonging))

	pkgParts := strings.Split(pkgPure, "/")
	version := pkgParts[len(pkgParts)-1]
	name := strings.Join(pkgParts[:len(pkgParts)-1], "/")
	name = strings.TrimLeft(name, "/")
	return name, version, belonging
}

func splitPackageNameAndVersionV6(pkg string) (string, string, string) {
	// sample input (lockfile 6.0+
	// 1. /@babel/code-frame@7.22.10
	// 2. /@babel/helper-create-regexp-features-plugin@7.22.9(@babel/core@7.22.10)
	// 3. /safe-buffer@5.1.2

	// Remove parentheses and content inside
	parts := strings.Split(pkg, "(")
	pkg = parts[0]

	atIndex := strings.LastIndex(pkg, "@")
	if atIndex == -1 {
		return "", "", ""
	}

	name := strings.TrimLeft(pkg[:atIndex], "/")
	version := pkg[atIndex+1:]

	// Extract extra content in parentheses
	extra := ""
	if len(parts) > 1 {
		extra = strings.TrimSuffix(parts[1], ")")
	}

	return name, version, extra
}

func processName(name string) (string, string, string) {
	nameWithoutAt := strings.TrimPrefix(name, "@")
	pkPath := name
	pkgNameParts := strings.Split(nameWithoutAt, "/")
	version := pkgNameParts[len(pkgNameParts)-1]

	return nameWithoutAt, pkPath, version
}

func appendNestedDependencies(deps []dependency) []dependency {
	allDeps := make([]dependency, 0)
	for _, d := range deps {
		allDeps = append(allDeps, d)
		if len(d.Dependencies) > 0 {
			for _, depD := range d.Dependencies {
				ar := strings.Split(strings.TrimSpace(depD), " ")
				name := strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(ar[0], "\""), "\""), "@")
				if name == "optionalDependencies:" {
					continue
				}

				version := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(ar[1]), "\""), "\"")
				if extractVersion(version) == "*" {
					continue
				}
				allDeps = append(allDeps, dependency{Name: name, Version: extractVersion(version)})
			}
		}
	}
	return allDeps
}
