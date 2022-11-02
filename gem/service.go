// SPDX-License-Identifier: Apache-2.0

package gem

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type (
	MetaVM struct {
		Name             string   `json:"name"`
		Downloads        int64    `json:"downloads"`
		Version          string   `json:"version"`
		VersionCreatedAt string   `json:"version_created_at"`
		VersionDownloads int64    `json:"version_downloads"`
		Platform         string   `json:"platform"`
		Authors          string   `json:"authors"`
		Info             string   `json:"info"`
		Licenses         []string `json:"licenses"`
		Metadata         struct {
			ChangeLogURI     string `json:"changelog_uri"`
			BugTrackerURI    string `json:"bug_tracker_uri"`
			SourceCodeURI    string `json:"source_code_uri"`
			DocumentationURI string `json:"documentation_uri"`
		} `json:"metadata"`
		Yanked         bool   `json:"yanked"`
		SHA            string `json:"sha"`
		ProjectURI     string `json:"project_uri"`
		GemURI         string `json:"gem_uri"`
		HomepageURI    string `json:"homepage_uri"`
		WikiURI        string `json:"wiki_uri"`
		DocumentURI    string `json:"documentation_uri"`
		MailingListURI string `json:"mailing_list_uri"`
		SourceCodeURI  string `json:"source_code_uri"`
		BugTrackerURI  string `json:"bug_tracker_uri"`
		ChangeLogURI   string `json:"changelog_uri"`
		FundingURI     string `json:"funding_uri"`
		Dependencies   struct {
			Development []struct {
				Name         string `json:"name"`
				Requirements string `json:"requirements"`
			} `json:"development"`
			Runtime []struct {
				Name         string `json:"name"`
				Requirements string `json:"requirements"`
			} `json:"runtime"`
		} `json:"dependencies"`
		BuiltAt         string `json:"built_at"`
		CreatedAt       string `json:"created_at"`
		Description     string `json:"description"`
		DownloadCount   int64  `json:"downloads_count"`
		Number          string `json:"number"`
		Summary         string `json:"summary"`
		RubyGemsVersion string `json:"rubygems_version"`
		RubyVersion     string `json:"ruby_version"`
		Prerelease      bool   `json:"prerelease"`
	}
	Service struct {
		request  *http.Request
		response *http.Response
		name     string
		err      error
	}
)

const (
	DefaultMethod       = "GET"
	DefaultURL          = "https://rubygems.org/api/v1/gems"
	DefaultResponseType = ".json"
)

func NewService(name string) (*Service, error) {
	url := fmt.Sprintf("%s/%s%s", DefaultURL, name, DefaultResponseType)
	request, err := http.NewRequest(DefaultMethod, url, nil)
	if err != nil {
		return nil, err
	}
	return &Service{
		request:  request,
		response: nil,
		name:     name,
		err:      nil,
	}, nil
}

func (service *Service) GetGem() (MetaVM, error) {
	var metadata MetaVM
	service.response, service.err = http.DefaultClient.Do(service.request)

	if service.err != nil {
		log.Printf("Failed to get gem from rubygems.org : %v\n", service.err)
		return MetaVM{}, service.err
	}
	defer func() {
		service.err = service.response.Body.Close()
		if service.err != nil {
			log.Printf("Failed to get gem from rubygems.org : %v\n", service.err)
		}
	}()

	service.err = json.NewDecoder(service.response.Body).Decode(&metadata)
	if service.err != nil {
		log.Printf("Failed to get gem from rubygems.org : %v\n", service.err)
		return MetaVM{}, service.err
	}

	return metadata, nil
}
