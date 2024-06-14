// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/opensbom-generator/parsers/meta"
	log "github.com/sirupsen/logrus"

	"github.com/opensbom-generator/parsers/internal/helper"
)

const pkgMetedataSeparator string = "---"

type GetPackageDetailsFunc = func(PackageName string) (string, error)

type MetadataDecoder struct {
	getPkgDetailsFunc GetPackageDetailsFunc
	pf                PypiPackageDataFactory
}

// NewMetadataDecoder ...
func NewMetadataDecoder(pkgDetailsFunc GetPackageDetailsFunc, pf PypiPackageDataFactory) *MetadataDecoder {
	return &MetadataDecoder{
		getPkgDetailsFunc: pkgDetailsFunc,
		pf:                pf,
	}
}

func SetMetadataValues(metadata *Metadata, datamap map[string]string) {
	metadata.Name = datamap[KeyName]
	metadata.Version = datamap[KeyVersion]
	metadata.Description = datamap[KeySummary]
	metadata.HomePage = datamap[KeyHomePage]
	metadata.Author = datamap[KeyAuthor]
	metadata.AuthorEmail = datamap[KeyAuthorEmail]
	metadata.License = datamap[KeyLicense]
	metadata.Location = datamap[KeyLocation]

	// Parsing "Requires"
	if len(datamap[KeyRequires]) != 0 {
		metadata.Modules = strings.Split(datamap[KeyRequires], ",")
		for i, v := range metadata.Modules {
			metadata.Modules[i] = strings.TrimSpace(v)
		}
	}
}

func ParseMetadata(metadata *Metadata, packagedetails string) {
	pkgDataMap := make(map[string]string, 10)
	resultlines := strings.Split(packagedetails, "\n")

	for _, resline := range resultlines {
		res := strings.Split(resline, ":")
		if len(res) <= 1 {
			continue
		}
		value := strings.TrimSpace(res[1])
		// If there are more elements, then concatenate the second element onwards
		// with a ":" in between
		if len(res) > 2 {
			for i := 2; i < len(res); i++ {
				value += ":" + res[i]
			}
		}
		pkgDataMap[strings.ToLower(res[0])] = value
	}
	SetMetadataValues(metadata, pkgDataMap)
}

func getAddionalMataDataInfo(metadata *Metadata) {
	metadata.ProjectURL = BuildProjectURL(metadata.Name)
	metadata.PackageURL = BuildPackageURL(metadata.Name)
	metadata.PackageReleaseURL = BuildPackageReleaseURL(metadata.Name, metadata.Version)
	metadata.PackageJSONURL = BuildPackageJSONURL(metadata.Name, metadata.Version)

	metadata.DistInfoPath = BuildDistInfoPath(metadata.Location, metadata.Name, metadata.Version)
	metadata.LocalPath = BuildLocalPath(metadata.Location, metadata.Name)
	metadata.LicensePath = BuildLicenseURL(metadata.DistInfoPath)
	metadata.MetadataPath = BuildMetadataPath(metadata.DistInfoPath)
	metadata.WheelPath = BuildWheelPath(metadata.DistInfoPath)
}

func (d *MetadataDecoder) BuildMetadata(pkgs []Packages) (map[string]Metadata, []Metadata, error) {
	metainfo := map[string]Metadata{}
	metaList := []Metadata{}
	pkgIndex := map[string]int{}

	var metadata *Metadata

	pkgNameList := ""
	for i, pkg := range pkgs {
		pkgNameList += pkg.Name + " "
		pkgIndex[strings.ToLower(pkg.Name)] = i
	}

	allpkgsmetadatastr, err := d.getPkgDetailsFunc(pkgNameList)
	if err != nil {
		return nil, nil, errorUnableToFetchPackageMetadata
	}

	// Metadata of all packages are separated by "---". Split all such occurrences and trim to remove leading \n

	a := regexp.MustCompile(pkgMetedataSeparator)
	eachpkgsmetadatastr := a.Split(allpkgsmetadatastr, -1)
	for i := range eachpkgsmetadatastr {
		eachpkgsmetadatastr[i] = strings.TrimSpace(eachpkgsmetadatastr[i])
	}

	for _, metadatastr := range eachpkgsmetadatastr {
		metadata = new(Metadata)
		ParseMetadata(metadata, metadatastr)
		getAddionalMataDataInfo(metadata)
		metadata.Root = pkgs[pkgIndex[strings.ToLower(metadata.Name)]].Root
		metadata.CPVersion = pkgs[pkgIndex[strings.ToLower(metadata.Name)]].CPVersion
		generator, tag, err := GetWheelDistributionInfo(metadata)
		if err != nil {
			log.Warnf("Wheel distribution info not found for `%s` package.", metadata.Name)
		}
		metadata.Generator = generator
		metadata.Tag = tag
		metaList = append(metaList, *metadata)
		metainfo[strings.ToLower(metadata.Name)] = *metadata
	}

	return metainfo, metaList, nil
}

func (d *MetadataDecoder) BuildModule(metadata Metadata) meta.Package {
	var module meta.Package

	// Prepare basic module info
	module.Root = metadata.Root
	module.Version = metadata.Version
	module.Name = metadata.Name
	module.Path = metadata.ProjectURL
	module.LocalPath = metadata.LocalPath
	module.PackageURL = metadata.PackageReleaseURL
	module.PackageHomePage = metadata.HomePage
	module.PackageComment = metadata.Description

	if (metadata.Root) && (len(metadata.HomePage) > 0) && metadata.HomePage != "None" {
		module.PackageURL = metadata.HomePage
	}

	pypiData, err := d.pf.GetPackageData(metadata.PackageJSONURL)
	if err != nil {
		log.Warnf("Unable to get `%s` package details from pypi.org", metadata.Name)
		if (len(metadata.HomePage) > 0) && (metadata.HomePage != "None") {
			module.PackageURL = metadata.HomePage
		}
	}

	// Prepare supplier contact
	if len(metadata.Author) > 0 && metadata.Author == "None" {
		metadata.Author, metadata.AuthorEmail = d.pf.GetMaintainerData(pypiData)
	}

	contactType := meta.Person
	if IsAuthorAnOrganization(metadata.Author, metadata.AuthorEmail) {
		contactType = meta.Organization
	}

	module.Supplier = meta.Supplier{
		Type:  contactType,
		Name:  metadata.Author,
		Email: metadata.AuthorEmail,
	}

	// Prepare checksum
	checksum := d.pf.GetChecksum(pypiData, metadata)
	module.Checksum = *checksum

	// Prepare download location
	downloadURL := d.pf.GetDownloadLocationFromPyPiPackageData(pypiData, metadata)
	module.PackageDownloadLocation = downloadURL
	if len(downloadURL) == 0 {
		if metadata.Root {
			module.PackageDownloadLocation = metadata.HomePage
		}
	}

	// Prepare licenses
	licensePkg, err := helper.GetLicenses(metadata.DistInfoPath)
	if err == nil {
		module.LicenseDeclared = helper.BuildLicenseDeclared(licensePkg.ID)
		module.LicenseConcluded = helper.BuildLicenseConcluded(licensePkg.ID)
		module.Copyright = helper.GetCopyright(licensePkg.ExtractedText)
		module.CommentsLicense = licensePkg.Comments
		if !helper.LicenseSPDXExists(licensePkg.ID) {
			licensePkg.ID = fmt.Sprintf("LicenseRef-%s", licensePkg.ID)
			licensePkg.ExtractedText = fmt.Sprintf("<text>%s</text>", licensePkg.ExtractedText)
			module.OtherLicense = append(module.OtherLicense, *licensePkg)
		}
	}

	// Prepare dependency module
	module.Packages = map[string]*meta.Package{}

	return module
}

func (d *MetadataDecoder) GetMetadataList(pkgs []Packages) (map[string]Metadata, []Metadata, error) {
	metainfo, metaList, err := d.BuildMetadata(pkgs)
	if err != nil {
		return nil, nil, err
	}

	return metainfo, metaList, nil
}

func (d *MetadataDecoder) ConvertMetadataToModules(pkgs []Packages, modules *[]meta.Package) (map[string]Metadata, error) {
	metainfo, metaList, err := d.GetMetadataList(pkgs)
	if err != nil {
		return nil, err
	}

	for _, metadata := range metaList {
		mod := d.BuildModule(metadata)
		*modules = append(*modules, mod)
	}
	return metainfo, nil
}

func BuildDependencyGraph(modules *[]meta.Package, pkgsMetadata *map[string]Metadata) error {
	moduleMap := map[string]meta.Package{}

	for _, module := range *modules {
		moduleMap[strings.ToLower(module.Name)] = module
	}

	for _, pkgmeta := range *pkgsMetadata {
		mod := moduleMap[strings.ToLower(pkgmeta.Name)]
		for _, modname := range pkgmeta.Modules {
			if depModule, ok := moduleMap[strings.ToLower(modname)]; ok {
				mod.Packages[depModule.Name] = &meta.Package{
					Version:          depModule.Version,
					Name:             depModule.Name,
					Path:             depModule.Path,
					LocalPath:        depModule.LocalPath,
					Supplier:         depModule.Supplier,
					PackageURL:       depModule.PackageURL,
					Checksum:         depModule.Checksum,
					PackageHomePage:  depModule.PackageHomePage,
					LicenseConcluded: depModule.LicenseConcluded,
					LicenseDeclared:  depModule.LicenseDeclared,
					CommentsLicense:  depModule.CommentsLicense,
					OtherLicense:     depModule.OtherLicense,
					Copyright:        depModule.Copyright,
					PackageComment:   depModule.PackageComment,
					Root:             depModule.Root,
				}
			} else {
				log.Warnf("Unable to find `%s` required by `%s`", modname, pkgmeta.Name)
			}
		}
	}

	return nil
}
