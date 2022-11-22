// SPDX-License-Identifier: Apache-2.0

package cargo

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

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

//counterfeiter:generate . cargoImplementation

// cargoImplementation implemnents the functionality to read cargo files and run the required commands
type cargoImplementation interface {
	GetCargoMetadata(string) (Metadata, error)
	GetCargoMetadataIfNeeded(*Mod, string) (*Metadata, error)
	ConvertPackagesToModulesList([]*Package) (map[string]*meta.Package, error)
	ConvertCargoPackageToMetaPackage(*Package) meta.Package
	ReadLockFile(string) (*LockFile, error)
	ReadConfig(string) (*Config, error)
	GetRootProjectName(string) (string, error)
	GetPackageDependencies(*Metadata, string) ([]*Package, error)
	GetRootModule(*Metadata, string) (meta.Package, error)
	PopulateDependencies(*Metadata, *meta.Package, bool, *map[string]*meta.Package) error
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

type LockFile struct {
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

type Config struct {
	Package         mainPackage
	RawDependencies map[string]interface{} `toml:"dependencies"`
	Dependencies    map[string]dependency  `toml:"omit"`
	Bin             []binaryData
}

func (di *defaultImplementation) ReadLockFile(path string) (*LockFile, error) {
	data, err := os.ReadFile(filepath.Join(path, lockFileName))
	if err != nil {
		return nil, fmt.Errorf("opening cargo lockfile: %w", err)
	}

	lf := &LockFile{}

	if err := toml.Unmarshal(data, lf); err != nil {
		return nil, fmt.Errorf("unmarshaling lockfile: %w", err)
	}

	return lf, nil
}

func (di *defaultImplementation) ReadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading toml configuration file: %w", err)
	}

	conf := &Config{
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

func (di *defaultImplementation) GetCargoMetadata(path string) (Metadata, error) {
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
	lockedData, err := di.ReadLockFile(path)
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

func (di *defaultImplementation) GetRootProjectName(path string) (string, error) {
	data, err := di.ReadConfig(filepath.Join(path, tomlFileName))
	if err != nil {
		return "", fmt.Errorf("parsing cargo toml configuration: %w", err)
	}
	return data.Package.Name, nil
}

// ConvertMetadataToModulesList gets a list of cargo metadata packages
// and converts it to our own metapackage
func (di *defaultImplementation) ConvertPackagesToModulesList(cargoPackages []*Package) (map[string]*meta.Package, error) {
	collection := map[string]*meta.Package{}
	for _, dep := range cargoPackages {
		module := di.ConvertCargoPackageToMetaPackage(dep)
		// Why this?! Is download location so important?
		if module.Name == "" || module.PackageDownloadLocation == "" {
			return nil, fmt.Errorf("incomplete information when converting package")
		}
		collection[module.Name] = &module
	}
	return collection, nil
}

// ConvertCargoPackageToModule converts a cargo metadata
// package to a meta.Package
func (di *defaultImplementation) ConvertCargoPackageToMetaPackage(dep *Package) meta.Package {
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

func (di *defaultImplementation) GetPackageDependencies(md *Metadata, rootName string) ([]*Package, error) {
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
func (di *defaultImplementation) GetCargoMetadataIfNeeded(m *Mod, path string) (*Metadata, error) {
	if m.cargoMetadata != nil {
		return m.cargoMetadata, nil
	}

	newMd, err := di.GetCargoMetadata(path)
	if err != nil {
		return nil, err
	}

	m.cargoMetadata = &newMd

	return m.cargoMetadata, nil
}

// populateDependencies
func (di *defaultImplementation) PopulateDependencies(
	md *Metadata, metaPackage *meta.Package, recurse bool, seen *map[string]*meta.Package,
) error {
	if seen == nil {
		seen = &map[string]*meta.Package{}
	}
	packages, err := di.GetPackageDependencies(md, metaPackage.Name)
	if err != nil {
		return fmt.Errorf("getting package dependencies: %w", err)
	}

	// Convert packages to metapackages
	metaPackages, err := di.ConvertPackagesToModulesList(packages)
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
			if err := di.PopulateDependencies(md, ptr, true, seen); err != nil {
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

func (di *defaultImplementation) GetRootModule(md *Metadata, path string) (meta.Package, error) {
	name, err := di.GetRootProjectName(path)
	if err != nil {
		return meta.Package{}, err
	}

	rootPackage := md.GetPackageByName(name)
	mod := convertCargoPackageToRootModule(*rootPackage)

	return mod, nil
}
