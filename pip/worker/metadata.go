// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"bufio"
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
)

const (
	ProjectURL          = "pypi.org/project"
	PackageURL          = "pypi.org/pypi"
	SitePackage         = "site-packages"
	PackageDistInfoPath = ".dist-info"
	PackageLicenseFile  = "LICENSE"
	PackageMetadataFie  = "METADATA"
	PackageWheelFie     = "WHEEL"

	// NOASSERTION constant
	NoAssertion = "NOASSERTION"

	KeyName        = "name"
	KeyVersion     = "version"
	KeySummary     = "summary"
	KeyHomePage    = "home-page"
	KeyAuthor      = "author"
	KeyAuthorEmail = "author-email"
	KeyLicense     = "license"
	KeyLocation    = "location"
	KeyRequires    = "requires"
)

var AuthorAnOrganizationKeywords = []string{"Authority", "Team", "Developers", "Services", "Foundation", "Software"}

type Packages struct {
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
	Location  string `json:"location,omitempty"`
	Installer string `json:"installer,omitempty"`
	Root      bool
	CPVersion string
}

type Metadata struct {
	CPVersion         string
	Root              bool
	Name              string
	Version           string
	Description       string
	ProjectURL        string
	PackageURL        string
	PackageJSONURL    string
	PackageReleaseURL string
	HomePage          string
	Author            string
	AuthorEmail       string
	License           string
	DistInfoPath      string
	LicensePath       string
	MetadataPath      string
	WheelPath         string
	Location          string
	LocalPath         string
	Modules           []string
	Generator         string
	Tag               string
}

var PythonVersion = map[string]string{
	"cp39": "Python 3.9",
	"cp38": "Python 3.8",
	"cp37": "Python 3.7",
	"cp36": "Python 3.6",
	"cp35": "Python 3.5",
	"cp34": "Python 3.4",
	"cp33": "Python 3.3",
	"cp32": "Python 3.2",
	"cp31": "Python 3.1",
	"cp30": "Python 3.0",
	"cp27": "Python 2.7",
	"cp26": "Python 2.6",
	"cp25": "Python 2.5",
	"cp24": "Python 2.4",
	"cp23": "Python 2.3",
	"cp22": "Python 2.2",
	"cp21": "Python 2.1",
	"cp20": "Python 2.0",
	"cp16": "Python 1.6",
	"cp15": "Python 1.5",
	"cp10": "Python 1.0",
}

func GetShortPythonVersion(version string) string {
	pythonVersion := "source"
	for k, v := range PythonVersion {
		if strings.Contains(version, v) {
			return k
		}
	}
	return pythonVersion
}

func LoadModules(data string, version string) []Packages {
	var _modules []Packages
	_ = json.Unmarshal([]byte(data), &_modules)
	for i, mod := range _modules {
		mod.CPVersion = version
		_modules[i] = mod
	}
	return _modules
}

func BuildProjectURL(name string) string {
	paths := []string{PackageURL, name}
	return path.Join(paths...)
}

func BuildPackageURL(name string) string {
	paths := []string{PackageURL, name}
	return path.Join(paths...)
}

func BuildPackageJSONURL(name string, version string) string {
	paths := []string{PackageURL, name, version, "json"}
	return path.Join(paths...)
}

func BuildPackageReleaseURL(name string, version string) string {
	paths := []string{PackageURL, name, version}
	return path.Join(paths...)
}

func BuildLocalPath(location string, name string) string {
	paths := []string{location, name}
	return path.Join(paths...)
}

func BuildDistInfoPath(location string, name string, version string) string {
	var distInfoPath string
	var exists bool
	var isSitePackage bool

	if strings.Contains(location, SitePackage) {
		isSitePackage = true
	}

	if isSitePackage {
		distInfoPath, exists = checkIfDistInfoPathExists(location, name, version)
		if exists {
			return distInfoPath
		}
		distInfoPath, _ = checkIfDistInfoPathExists(location, strings.ReplaceAll(name, "-", "_"), version)
	} else {
		distInfoPath = location
	}
	return distInfoPath
}

func checkIfDistInfoPathExists(location string, name string, version string) (string, bool) {
	var distInfoPath string

	packageName := name + "-" + version
	packageMetadata := packageName + PackageDistInfoPath
	paths := []string{location, packageMetadata}

	distInfoPath = path.Join(paths...)

	return distInfoPath, helper.Exists(distInfoPath)
}

func BuildLicenseURL(distInfoLocation string) string {
	paths := []string{distInfoLocation, PackageLicenseFile}
	return path.Join(paths...)
}

func BuildMetadataPath(distInfoLocation string) string {
	paths := []string{distInfoLocation, PackageMetadataFie}
	return path.Join(paths...)
}

func BuildWheelPath(distInfoLocation string) string {
	paths := []string{distInfoLocation, PackageWheelFie}
	return path.Join(paths...)
}

func SetMetadataToNoAssertion(metadata *Metadata, packagename string) {
	metadata.Name = packagename
	metadata.Version = NoAssertion
	metadata.Description = NoAssertion
	metadata.HomePage = NoAssertion
	metadata.Author = NoAssertion
	metadata.AuthorEmail = NoAssertion
	metadata.License = NoAssertion
	metadata.LocalPath = NoAssertion
	metadata.Modules = []string{}
}

func IsAuthorAnOrganization(author string, authoremail string) bool {
	// If both Author and Author-Email are defined as "None", we assume it as Organization
	if (strings.Compare(strings.ToLower(author), "none") == 0) &&
		(strings.Compare(strings.ToLower(authoremail), "none") == 0) {
		return true
	}

	result := false
	// If Author fields has any one of the keywords, we assume it as Organization
	for _, v := range AuthorAnOrganizationKeywords {
		if strings.Contains(strings.ToLower(author), strings.ToLower(v)) {
			result = true
			break
		}
	}
	return result
}

func GetWheelDistributionInfo(metadata *Metadata) (string, string, error) {
	if !helper.Exists(metadata.WheelPath) {
		return "", "", errorWheelFileNotFound
	}

	file, err := os.Open(metadata.WheelPath)
	if err != nil {
		return "", "", errorUnableToOpenWheelFile
	}
	defer file.Close()

	generator := ""
	tag := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		res := strings.Split(scanner.Text(), ":")
		if strings.Compare(strings.ToLower(res[0]), "generator") == 0 {
			gen := strings.Split(strings.TrimSpace(res[1]), " ")
			generator = strings.TrimSpace(gen[0])
		}
		if strings.Compare(strings.ToLower(res[0]), "tag") == 0 {
			tag = strings.TrimSpace(res[1])
		}
	}

	return generator, tag, nil
}
