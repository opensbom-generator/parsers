// SPDX-License-Identifier: Apache-2.0

package swift

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
)

func (description *PackageDescription) Module() *meta.Package {
	mod := &meta.Package{}

	mod.Name = description.Name
	mod.Root = true
	mod.LocalPath = description.Path

	setLicense(mod, description.Path)  // nolint:errcheck
	setCheckSum(mod, description.Path) // nolint:errcheck
	setVersion(mod, description.Path)  // nolint:errcheck

	return mod
}

func (dep *PackageDependency) Module() *meta.Package {
	mod := &meta.Package{}
	mod.Name = dep.Name
	mod.PackageURL = strings.TrimSuffix(dep.URL, ".git")

	if strings.HasSuffix(dep.URL, ".git") {
		if strings.HasPrefix(dep.URL, "http") ||
			strings.HasPrefix(dep.URL, "ssh") ||
			strings.HasPrefix(dep.URL, "git@") {
			mod.PackageDownloadLocation = "git+" + dep.URL
		}
	}

	mod.Version = dep.Version
	mod.LocalPath = dep.Path

	setLicense(mod, dep.Path)  // nolint:errcheck
	setCheckSum(mod, dep.Path) // nolint:errcheck

	return mod
}

func setLicense(mod *meta.Package, path string) error {
	licensePkg, err := helper.GetLicenses(path)
	if err != nil {
		return err
	}

	mod.LicenseDeclared = helper.BuildLicenseDeclared(licensePkg.ID)
	mod.LicenseConcluded = helper.BuildLicenseConcluded(licensePkg.ID)
	if !helper.LicenseSPDXExists(licensePkg.ID) {
		licensePkg.ID = fmt.Sprintf("LicenseRef-%s", licensePkg.ID)
		mod.OtherLicense = append(mod.OtherLicense, *licensePkg)
	}
	mod.Copyright = helper.GetCopyright(licensePkg.ExtractedText)
	mod.CommentsLicense = licensePkg.Comments

	return nil
}

func setVersion(mod *meta.Package, path string) error {
	cmd := exec.Command("git", "describe", "--tags", "--exact-match")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		version := scanner.Text()

		// semver requires a "v" prefix
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}

		if semver.IsValid(version) {
			mod.Version = version[1:] // remove the "v" prefix
			break
		}
	}

	return nil
}

func setCheckSum(mod *meta.Package, path string) error {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	if len(output) > 0 {
		mod.Checksum = meta.Checksum{
			Algorithm: meta.HashAlgoSHA1, // FIXME: derive from git
			Value:     string(output),
		}
	}

	return nil
}
