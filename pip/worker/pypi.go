// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
)

var (
	errorPypiCouldNotFetchPkgData = errors.New("could not fetch package data from PyPI")
)

type PypiPackageData struct {
	Info PypiPackageInfo       `json:"info"`
	Urls []PypiPackageDistInfo `json:"urls"`
}

type PypiPackageInfo struct {
	Author                 string   `json:"author"`
	AuthorEmail            string   `json:"author_email"`
	BugTrackURL            string   `json:"bugtrack_url"`
	Classifiers            []string `json:"classifiers"`
	Description            string   `json:"description"`
	DescriptionContentType string   `json:"description_content_type"`
	DocsURL                string   `json:"docs_url"`
	DownloadURL            string   `json:"download_url"`
	HomePageURL            string   `json:"home_page"`
	Keywords               string   `json:"keywords"`
	License                string   `json:"license"`
	Maintainer             string   `json:"maintainer"`
	MaintainerEmail        string   `json:"maintainer_email"`
	Name                   string   `json:"name"`
	PackageURL             string   `json:"package_url"`
	Platform               string   `json:"platform"`
	ProjectURL             string   `json:"project_url"`
	ReleaseURL             string   `json:"release_url"`
	RequiresDist           []string `json:"requires_dist"`
	RequiresPython         string   `json:"requires_python"`
	Summary                string   `json:"summary"`
	Version                string   `json:"version"`
	Yanked                 bool     `json:"yanked"`
	YankedReason           string   `json:"yanked_reason"`
}

type PypiPackageDistInfo struct {
	CommentText    string      `json:"comment_text"`
	Digests        DigestTypes `json:"digests"`
	Downloads      int         `json:"downloads"`
	Filename       string      `json:"filename"`
	HasSig         bool        `json:"has_sig"`
	MD5Digest      string      `json:"md5_digest"`
	PackageType    string      `json:"packagetype"`
	PythonVersion  string      `json:"python_version"`
	RequiresPython string      `json:"requires_python"`
	URL            string      `json:"url"`
	Yanked         bool        `json:"yanked"`
	YankedReason   string      `json:"yanked_reason"`
}

type DigestTypes struct {
	MD5    string `json:"md5"`
	SHA256 string `json:"sha256"`
}

// HashAlgoPickOrder Order in which we want to pick the package digest
var HashAlgoPickOrder []meta.HashAlgorithm = []meta.HashAlgorithm{
	meta.HashAlgoSHA512,
	meta.HashAlgoSHA384,
	meta.HashAlgoSHA256,
	meta.HashAlgoSHA224,
	meta.HashAlgoSHA1,
	meta.HashAlgoMD6,
	meta.HashAlgoMD5,
	meta.HashAlgoMD4,
	meta.HashAlgoMD2,
}

type pypiPackageDataFactory struct {
	client *helper.Client
}

type PypiPackageDataFactory interface {
	GetPackageData(packageJSONURL string) (PypiPackageData, error)
	GetMaintainerData(pkgData PypiPackageData) (string, string)
	GetChecksum(pkgData PypiPackageData, metadata Metadata) *meta.Checksum
	GetDownloadLocationFromPyPiPackageData(pkgData PypiPackageData, metadata Metadata) string
}

// NewPypiPackageDataFactory ...
func NewPypiPackageDataFactory(client *helper.Client) PypiPackageDataFactory {
	return &pypiPackageDataFactory{
		client: client,
	}
}

func (pf *pypiPackageDataFactory) GetPackageData(packageJSONURL string) (PypiPackageData, error) {
	packageInfo := PypiPackageData{}

	packageJSONURL = strings.Replace(packageJSONURL, "pypi.org", "", 1)
	response, err := pf.client.HTTP.Get(fmt.Sprintf("%s%s", pf.client.BaseURL, packageJSONURL))
	if err != nil {
		return packageInfo, err
	}

	if response.StatusCode != http.StatusOK {
		return packageInfo, errorPypiCouldNotFetchPkgData
	}
	defer response.Body.Close()

	jsondata, _ := io.ReadAll(response.Body)

	err = json.Unmarshal(jsondata, &packageInfo)
	if err != nil {
		return packageInfo, err
	}
	return packageInfo, nil
}

func (pf *pypiPackageDataFactory) GetMaintainerData(pkgData PypiPackageData) (string, string) {
	var name string
	var email string
	if len(pkgData.Info.Maintainer) > 0 {
		name = strings.TrimSpace(pkgData.Info.Maintainer)
	}
	if len(pkgData.Info.MaintainerEmail) > 0 {
		email = strings.TrimSpace(pkgData.Info.MaintainerEmail)
	}
	return name, email
}

func (pf *pypiPackageDataFactory) GetChecksum(pkgData PypiPackageData, metadata Metadata) *meta.Checksum {
	checksum := meta.Checksum{
		Algorithm: meta.HashAlgoSHA1,
		Content:   []byte(pkgData.Info.Name),
	}

	for _, packageDistInfo := range pkgData.Urls {
		distInfo, status := getPackageBDistWheelInfo(packageDistInfo, metadata.Generator, metadata.Tag, metadata.CPVersion)
		if status {
			algo, value := getHighestOrderHashData(distInfo.Digests)
			checksum.Algorithm = algo
			checksum.Value = value
			return &checksum
		}

		distInfo, status = getPackageSDistInfo(packageDistInfo, "sdist")
		if status {
			algo, value := getHighestOrderHashData(distInfo.Digests)
			checksum.Algorithm = algo
			checksum.Value = value
			return &checksum
		}
	}

	return &checksum
}

func (pf *pypiPackageDataFactory) GetDownloadLocationFromPyPiPackageData(pkgData PypiPackageData, metadata Metadata) string {
	for _, packageDistInfo := range pkgData.Urls {
		distInfo, status := getPackageBDistWheelInfo(packageDistInfo, metadata.Generator, metadata.Tag, metadata.CPVersion)
		if status {
			return distInfo.URL
		}

		distInfo, status = getPackageSDistInfo(packageDistInfo, "sdist")
		if status {
			return distInfo.URL
		}
	}

	return ""
}

func getPackageBDistWheelInfo(distInfo PypiPackageDistInfo, generator string,
	tag string, cpVersion string) (PypiPackageDistInfo, bool) {
	PackageType := strings.EqualFold(distInfo.PackageType, generator)
	Tag := strings.Contains(strings.ToLower(distInfo.Filename), strings.ToLower(tag))
	CPVersion := strings.EqualFold(distInfo.PythonVersion, cpVersion)
	Py2Py3 := strings.Contains(strings.ToLower("py2.py3"), strings.ToLower(distInfo.PythonVersion))

	status := false

	if PackageType && Tag && (CPVersion || Py2Py3) {
		status = true
	}

	return distInfo, status
}

func getPackageSDistInfo(distInfo PypiPackageDistInfo, generator string) (PypiPackageDistInfo, bool) {
	PackageType := strings.EqualFold(distInfo.PackageType, generator)
	Source := strings.EqualFold(distInfo.PythonVersion, "source")

	status := false

	if PackageType && Source {
		status = true
	}

	return distInfo, status
}

func getHighestOrderHashData(digests DigestTypes) (meta.HashAlgorithm, string) {
	var algoType meta.HashAlgorithm
	var digestValue string

	v := reflect.ValueOf(digests)
	for _, algo := range HashAlgoPickOrder {
		f := v.FieldByName(string(algo))
		if f.IsValid() {
			algoType = algo
			digestValue = f.String()
			return algoType, digestValue
		}
	}

	return algoType, digestValue
}
