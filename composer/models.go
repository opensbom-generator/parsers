// SPDX-License-Identifier: Apache-2.0

package composer

type LockFile struct {
	Packages    []LockPackage
	PackagesDev []LockPackage `json:"packages-dev"`
}

type LockPackage struct {
	Name        string
	Version     string
	Type        string
	Dist        LockPackageDist
	License     []string
	Description string
	Source      LockPackageSource
	Authors     []LockPackageAuthor
	Homepage    string
}
type LockPackageAuthor struct {
	Name  string
	Email string
}

type LockPackageSource struct {
	Type      string
	URL       string
	Reference string
}

type LockPackageDist struct {
	Type      string
	URL       string
	Reference string
	Shasum    string
}

type ProjectInfo struct {
	Name        string
	Description string
	Versions    []string
}

type TreeList struct {
	Installed []TreeComponent
}
type TreeComponent struct {
	Name        string
	Version     string
	Description string
	Requires    []TreeComponent
}

type JSONObject struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Homepage    string   `json:"homepage"`
	License     string   `json:"license"`
	Authors     []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"authors"`
}

type PackageJSONObject struct {
	Name       string `json:"name"`
	Title      string `json:"title"`
	Version    string `json:"version"`
	Homepage   string `json:"homepage"`
	Repository struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"repository"`
}
