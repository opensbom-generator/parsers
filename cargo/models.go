// SPDX-License-Identifier: Apache-2.0

package cargo

type Metadata struct {
	WorkspaceRoot   string    `json:"workspace_root"`
	Version         int64     `json:"version"`
	TargetDirectory string    `json:"target_directory"`
	Packages        []Package `json:"packages"`
}

type Package struct {
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	ID           string              `json:"id"`
	Description  string              `json:"description"`
	Source       string              `json:"source"`
	Dependencies []PackageDependency `json:"dependencies"`
	ManifestPath string              `json:"manifest_path"`
	Authors      []string            `json:"authors"`
	Repository   string              `json:"repository"`
	Homepage     string              `json:"homepage"`
	License      string              `json:"license"`

	Checksum string `json:"-"` // This data bit gets read from the lockfile
}

type PackageDependency struct {
	Name                string        `json:"name"`
	Source              string        `json:"source"`
	Req                 string        `json:"req"`
	Kind                interface{}   `json:"kind"`
	Rename              interface{}   `json:"rename"`
	Optional            bool          `json:"optional"`
	UsesDefaultFeatures bool          `json:"uses_default_features"`
	Features            []interface{} `json:"features"`
	Target              interface{}   `json:"target"`
	Registry            interface{}   `json:"registry"`
}

func (md *Metadata) GetPackageByName(name string) *Package {
	for _, p := range md.Packages {
		if p.Name == name {
			return &p
		}
	}
	return nil
}
