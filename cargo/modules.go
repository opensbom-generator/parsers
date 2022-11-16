// SPDX-License-Identifier: Apache-2.0

package cargo

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
)

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
	md, err := m.impl.getCargoMetadataIfNeeded(m, path)
	if err != nil {
		return meta.Package{}, fmt.Errorf("getting cargo metadata: %w", err)
	}
	return m.impl.getRootModule(md, path)
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
