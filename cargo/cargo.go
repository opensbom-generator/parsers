// SPDX-License-Identifier: Apache-2.0

package cargo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/release-utils/command"
)

const (
	Cmd               = "cargo"
	VersionArg        = "--version"
	ModulesCmd        = "cargo metadata --format-version=1"
	RootModuleNameCmd = "cargo pkgid"

	tomlFileName = "Cargo.toml"
	lockFileName = "Cargo.lock"
)

// Implementation to fake later
type cargoImplementation interface {
	getCargoMetadata(string) (Metadata, error)
	getCargoMetadataIfNeeded(*Mod, string) (*Metadata, error)
	convertPackagesToModulesList([]*Package) (map[string]*meta.Package, error)
	convertCargoPackageToMetaPackage(*Package) meta.Package
	readLockFile(string) (*lockFile, error)
	readConfig(string) (*config, error)
	getRootProjectName(string) (string, error)
	getPackageDependencies(*Metadata, string) ([]*Package, error)
	getRootModule(*Metadata, string) (meta.Package, error)
	populateDependencies(*Metadata, *meta.Package, bool, *map[string]*meta.Package) error
}

type defaultImplementation struct{}

type lockedPackage struct {
	Name         string
	Version      string
	Source       string
	Checksum     string
	Dependencies []string `toml:"dependencies"`
	Packages     map[string]*lockedPackage
}

type lockFile struct {
	Version  int
	Packages []lockedPackage `toml:"package"`
}

type mainPackage struct {
	Name    string
	Version string
	Edition string
}

type dependency struct {
	Name    string
	Version string
}

type binaryData struct {
	Name string
	Path string
}

type config struct {
	Package         mainPackage
	RawDependencies map[string]interface{} `toml:"dependencies"`
	Dependencies    map[string]dependency  `toml:"omit"`
	Bin             []binaryData
}

func (di *defaultImplementation) readLockFile(path string) (*lockFile, error) {
	data, err := os.ReadFile(filepath.Join(path, lockFileName))
	if err != nil {
		return nil, fmt.Errorf("opening cargo lockfile: %w", err)
	}

	lf := &lockFile{}

	if err := toml.Unmarshal(data, lf); err != nil {
		return nil, fmt.Errorf("unmarshaling lockfile: %w", err)
	}

	return lf, nil
}

func (di *defaultImplementation) readConfig(path string) (*config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading toml configuration file: %w", err)
	}

	conf := &config{
		Package:         mainPackage{},
		RawDependencies: map[string]interface{}{},
		Dependencies:    map[string]dependency{},
		Bin:             []binaryData{},
	}

	if err := toml.Unmarshal(data, conf); err != nil {
		return nil, fmt.Errorf("unmarshaling toml config file: %w", err)
	}

	// Assign the read dependencies
	for name, rawData := range conf.RawDependencies {
		dep := dependency{}
		dep.Name = name
		if _, ok := rawData.(string); ok {
			dep.Version = rawData.(string)
		}

		if table, ok := rawData.(map[string]interface{}); ok {
			if i, ok := table["version"]; ok {
				dep.Version = i.(string)
			}
		}
		conf.Dependencies[dep.Name] = dep
	}
	return conf, nil
}

func (di *defaultImplementation) getCargoMetadata(path string) (Metadata, error) {
	// to be assembled from the output of:
	// rustc --print cfg
	// using target_arch target_vendor target_os target_env
	var cargoMetadata Metadata
	logrus.Infof("running cargo --metadata")
	cmdArgs := []string{
		"metadata",
		"--filter-platform=x86_64-unknown-linux-gnu", // TODO: Detect effective platform or option
	}
	output, err := command.NewWithWorkDir(
		path, string(Cmd), cmdArgs...).RunSilentSuccessOutput()
	if err != nil {
		return cargoMetadata, fmt.Errorf("running cargo metadata: %w", err)
	}

	if err := json.Unmarshal([]byte(output.Output()), &cargoMetadata); err != nil {
		return cargoMetadata, fmt.Errorf("decoding cargo metadata: %w", err)
	}

	// Get the locked datato get the hashes
	lockedData, err := di.readLockFile(path)
	if err != nil {
		return cargoMetadata, fmt.Errorf("getting locked data: %w", err)
	}

	// Populate the checksums
	catalog := map[string]string{}
	for _, p := range lockedData.Packages {
		catalog[p.Name+":"+p.Version] = p.Checksum
	}

	for i := range cargoMetadata.Packages {
		if cs, ok := catalog[cargoMetadata.Packages[i].Name+":"+cargoMetadata.Packages[i].Version]; ok {
			cargoMetadata.Packages[i].Checksum = cs
		}
	}

	logrus.Infof("Got data describing %d packages", len(cargoMetadata.Packages))
	return cargoMetadata, nil
}

func (di *defaultImplementation) getRootProjectName(path string) (string, error) {
	data, err := di.readConfig(filepath.Join(path, tomlFileName))
	if err != nil {
		return "", fmt.Errorf("parsing cargo toml configuration: %w", err)
	}
	return data.Package.Name, nil
}

// convertMetadataToModulesList gets a list of cargo metadata packages
// and converts it to our own metapackage
func (di *defaultImplementation) convertPackagesToModulesList(cargoPackages []*Package) (map[string]*meta.Package, error) {
	collection := map[string]*meta.Package{}
	for _, dep := range cargoPackages {
		module := di.convertCargoPackageToMetaPackage(dep)
		// Why this?! Is download location so important?
		if module.Name == "" || module.PackageDownloadLocation == "" {
			return nil, fmt.Errorf("incomplete information when converting package")
		}
		collection[module.Name] = &module
	}
	return collection, nil
}

// convertCargoPackageToModule converts a cargo metadata
// package to a meta.Package
func (di *defaultImplementation) convertCargoPackageToMetaPackage(dep *Package) meta.Package {
	localPath := convertToLocalPath(dep.ManifestPath)
	supplier := getPackageSupplier(dep.Authors, dep.Name)

	// We know where to get crates packages
	downloadURL := ""
	if dep.Source == "registry+https://github.com/rust-lang/crates.io-index" {
		downloadURL = fmt.Sprintf("https://crates.io/api/v1/crates/%s/%s/download", dep.Name, dep.Version)
	}

	module := meta.Package{
		Version:    dep.Version,
		Name:       dep.Name,
		Root:       false,
		PackageURL: formatPackageURL(*dep),
		Checksum: meta.Checksum{
			Algorithm: meta.HashAlgoSHA256,
			Value:     dep.Checksum,
		},
		LocalPath:               localPath,
		PackageHomePage:         dep.Homepage,
		Supplier:                supplier,
		PackageDownloadLocation: downloadURL,
		Packages:                map[string]*meta.Package{},
	}

	licensePkg, err := helper.GetLicenses(localPath)
	if err == nil {
		module.LicenseDeclared = helper.BuildLicenseDeclared(licensePkg.ID)
		module.LicenseConcluded = helper.BuildLicenseConcluded(licensePkg.ID)
		module.Copyright = helper.GetCopyright(licensePkg.ExtractedText)
		module.CommentsLicense = licensePkg.Comments
	} else if dep.License != "" {
		module.LicenseDeclared = dep.License
		module.LicenseConcluded = dep.License
	}

	return module
}

func (di *defaultImplementation) getPackageDependencies(md *Metadata, rootName string) ([]*Package, error) {
	// First get the names of the deps
	rootPackage := md.GetPackageByName(rootName)
	if rootPackage == nil {
		return nil, fmt.Errorf("unable to find %s in cargo packages", rootName)
	}

	// Search the packages
	packages := []*Package{}
	for _, dep := range rootPackage.Dependencies {
		depPackage := md.GetPackageByName(dep.Name)
		if depPackage == nil {
			continue
		}
		packages = append(packages, depPackage)
	}
	logrus.Debugf("Package %s has %d dependencies", rootName, len(packages))
	return packages, nil
}

// getCargoMetadataIfNeeded checks if we need to load metadata or not
func (di *defaultImplementation) getCargoMetadataIfNeeded(m *Mod, path string) (*Metadata, error) {
	if m.cargoMetadata != nil {
		return m.cargoMetadata, nil
	}

	newMd, err := di.getCargoMetadata(path)
	if err != nil {
		return nil, err
	}

	m.cargoMetadata = &newMd

	return m.cargoMetadata, nil
}

// populateDependencies
func (di *defaultImplementation) populateDependencies(
	md *Metadata, metaPackage *meta.Package, recurse bool, seen *map[string]*meta.Package,
) error {
	if seen == nil {
		seen = &map[string]*meta.Package{}
	}
	packages, err := di.getPackageDependencies(md, metaPackage.Name)
	if err != nil {
		return fmt.Errorf("getting package dependencies: %w", err)
	}

	// Convert packages to metapackages
	metaPackages, err := di.convertPackagesToModulesList(packages)
	if err != nil {
		return fmt.Errorf("converting cargo packages: %w", err)
	}
	if len(metaPackages) != len(packages) {
		logrus.Warnf(
			"Number of converted metapackages don't match cargo packages (%d vs %d)",
			len(packages), len(metaPackages),
		)
	}

	if !recurse {
		metaPackage.Packages = metaPackages
		return nil
	}

	(*seen)[metaPackage.Name+":"+metaPackage.Version] = metaPackage

	// get deps of deps
	for _, ptr := range metaPackages {
		if _, ok := (*seen)[ptr.Name+":"+ptr.Version]; !ok {
			if err := di.populateDependencies(md, ptr, true, seen); err != nil {
				return fmt.Errorf("getting dependencies of %s: %w", ptr.Name, err)
			}
		} else {
			ptr.Packages = (*seen)[ptr.Name+":"+ptr.Version].Packages
		}
		(*seen)[ptr.Name+":"+ptr.Version] = ptr
	}

	metaPackage.Packages = metaPackages
	return nil
}

func (di *defaultImplementation) getRootModule(md *Metadata, path string) (meta.Package, error) {
	name, err := di.getRootProjectName(path)
	if err != nil {
		return meta.Package{}, err
	}

	rootPackage := md.GetPackageByName(name)
	mod := convertCargoPackageToRootModule(*rootPackage)

	return mod, nil
}
