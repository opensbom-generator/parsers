// SPDX-License-Identifier: Apache-2.0

package cargo

import (
	"encoding/json"
	"net/mail"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
)

func addDepthModules(modules []meta.Package, cargoPackages []Package) {
	moduleMap := map[string]meta.Package{}
	moduleIndex := map[string]int{}
	for idx, module := range modules {
		moduleMap[module.Name] = module
		moduleIndex[module.Name] = idx
	}

	for _, cargoPackage := range cargoPackages {
		rootLevelName := cargoPackage.Name
		if rootLevelName == "" {
			continue
		}

		rootModuleIndex, ok := moduleIndex[rootLevelName]
		if !ok {
			continue
		}

		cargoDependencies := cargoPackage.Dependencies
		if len(cargoDependencies) == 0 {
			continue
		}

		for _, cargoDep := range cargoDependencies {
			subModuleName := cargoDep.Name
			if subModuleName == "" {
				continue
			}

			subModule, ok := moduleMap[subModuleName]
			if !ok {
				continue
			}

			modules[rootModuleIndex].Packages[subModuleName] = &meta.Package{
				Name:             subModule.Name,
				Version:          subModule.Version,
				Path:             subModule.Path,
				LocalPath:        subModule.LocalPath,
				Supplier:         subModule.Supplier,
				PackageURL:       subModule.PackageURL,
				Checksum:         subModule.Checksum,
				PackageHomePage:  subModule.PackageHomePage,
				LicenseConcluded: subModule.LicenseConcluded,
				LicenseDeclared:  subModule.LicenseDeclared,
				CommentsLicense:  subModule.CommentsLicense,
				OtherLicense:     subModule.OtherLicense,
				Copyright:        subModule.Copyright,
				PackageComment:   subModule.PackageComment,
				Root:             subModule.Root,
			}
		}
	}
}

func convertMetadataToModulesList(cargoPackages []Package) []meta.Package {
	collection := make([]meta.Package, len(cargoPackages))
	for _, dep := range cargoPackages {
		module := convertCargoPackageToModule(dep)
		if module.Name == "" || module.PackageDownloadLocation == "" {
			continue
		}

		collection = append(collection, module)
	}

	return collection
}

func convertCargoPackageToModule(dep Package) meta.Package {
	localPath := convertToLocalPath(dep.ManifestPath)
	supplier := getPackageSupplier(dep.Authors, dep.Name)
	donwloadURL := getPackageDownloadLocation(dep)

	module := meta.Package{
		Version:    dep.Version,
		Name:       dep.Name,
		Root:       false,
		PackageURL: formatPackageURL(dep),
		Checksum: meta.Checksum{
			Algorithm: meta.HashAlgoSHA1,
			Value:     readCheckSum(dep.ID),
		},
		LocalPath:               localPath,
		PackageHomePage:         dep.Homepage,
		Supplier:                supplier,
		PackageDownloadLocation: donwloadURL,
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

func getPackageDownloadLocation(dep Package) string {
	if dep.Repository != "" {
		return dep.Repository
	}

	source := strings.ReplaceAll(dep.Source, "registry+", "")
	if source != "" {
		return source
	}

	if dep.Homepage != "" {
		return dep.Homepage
	}

	return ""
}

func getPackageSupplier(authors []string, defaultValue string) meta.Supplier {
	if len(authors) == 0 {
		return meta.Supplier{
			Name: defaultValue,
		}
	}

	var supplier meta.Supplier

	mainAuthor := authors[0]
	author, _ := mail.ParseAddress(mainAuthor)

	if author != nil {
		supplier = meta.Supplier{
			Name:  author.Name,
			Email: author.Address,
			Type:  meta.Person,
		}
	}

	if supplier.Email == "" {
		supplier.Type = meta.Organization
	}

	if supplier.Name == "" {
		if mainAuthor != "" {
			supplier.Name = mainAuthor
		} else {
			supplier.Name = defaultValue
		}
	}

	return supplier
}

func (m *Mod) getRootModule(path string) (meta.Package, error) {
	name, err := m.getRootProjectName(path)
	if err != nil {
		return meta.Package{}, err
	}

	cargoMetadata, err := m.getCargoMetadata(path)
	if err != nil {
		return meta.Package{}, err
	}

	packages := cargoMetadata.Packages
	rootPackage, _ := findPackageByName(packages, name)
	mod := convertCargoPackageToRootModule(rootPackage)
	return mod, nil
}

func convertCargoPackageToRootModule(dep Package) meta.Package {
	localPath := convertToLocalPath(dep.ManifestPath)

	module := meta.Package{
		Version:    dep.Version,
		Name:       dep.Name,
		Root:       true,
		PackageURL: formatPackageURL(dep),
		Checksum: meta.Checksum{
			Algorithm: meta.HashAlgoSHA1,
			Value:     readCheckSum(dep.ID),
		},
		LocalPath:               localPath,
		PackageHomePage:         removeURLProtocol(dep.Homepage),
		Supplier:                getPackageSupplier(dep.Authors, dep.Name),
		Packages:                map[string]*meta.Package{},
		PackageDownloadLocation: dep.Repository,
	}

	licensePkg, err := helper.GetLicenses(localPath)
	if err == nil {
		module.LicenseDeclared = helper.BuildLicenseDeclared(licensePkg.ID)
		module.LicenseConcluded = helper.BuildLicenseConcluded(licensePkg.ID)
		module.Copyright = helper.GetCopyright(licensePkg.ExtractedText)
		module.CommentsLicense = licensePkg.Comments
	}

	return module
}

func convertToLocalPath(manifestPath string) string {
	localPath := strings.ReplaceAll(manifestPath, "/Cargo.toml", "")
	return localPath
}

func getDefaultPackageURL(dep Package) string {
	if dep.Homepage != "" {
		return dep.Homepage
	}

	if dep.Source != "" {
		return dep.Source
	}

	return dep.Repository
}

func formatPackageURL(dep Package) string {
	URL := getDefaultPackageURL(dep)
	URL = removeURLProtocol(URL)
	URL = removeRegisrySuffix(URL)

	return URL
}

func (m *Mod) getCargoMetadata(path string) (Metadata, error) {
	if m.cargoMetadata.WorkspaceRoot != "" {
		return m.cargoMetadata, nil
	}

	buff, _ := m.runTask(ModulesCmd, path)
	defer buff.Reset()

	var cargoMetadata Metadata
	if err := json.NewDecoder(buff).Decode(&cargoMetadata); err != nil {
		return Metadata{}, err
	}
	m.cargoMetadata = cargoMetadata

	return m.cargoMetadata, nil
}

func (m *Mod) getRootProjectName(path string) (string, error) {
	err := m.buildCmd(RootModuleNameCmd, path)
	if err != nil {
		return "", err
	}

	pckidRoot, err := m.command.Output()
	if err != nil {
		return "", erroRootPackageInformation
	}
	parts := strings.Split(pckidRoot, "/")
	lastpart := parts[len(parts)-1]
	lastpart = strings.ReplaceAll(lastpart, "\n", "")

	rootNameParts := strings.Split(lastpart, "#")
	name := rootNameParts[0]

	return name, nil
}

func findPackageByName(packages []Package, name string) (Package, bool) {
	for _, mod := range packages {
		if mod.Name == name {
			return mod, true
		}
	}

	return Package{}, false
}
