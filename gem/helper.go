// SPDX-License-Identifier: Apache-2.0

package gem

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/opensbom-generator/parsers/meta"
	log "github.com/sirupsen/logrus"

	"github.com/opensbom-generator/parsers/internal/helper"
)

var (
	dependencies = []Package{}
	lines        = []Line{}

	Required = map[string]bool{
		"s.name":                          true,
		"spec.name":                       true,
		"s.license":                       true,
		"spec.license":                    true,
		"s.licenses":                      true,
		"spec.licenses":                   true,
		"s.email":                         true,
		"spec.email":                      true,
		"s.homepage":                      true,
		"spec.homepage":                   true,
		"s.authors":                       true,
		"spec.authors":                    true,
		"s.summary":                       true,
		"spec.summary":                    true,
		"s.description":                   true,
		"spec.description":                true,
		"s.add_runtime_dependency":        true,
		"spec.add_runtime_dependency":     true,
		"s.add_development_dependency":    true,
		"spec.add_development_dependency": true,
		"s.add_dependency":                true,
		"spec.add_dependency":             true,
		"s.rubygems_version":              true,
		"spec.rubygems_version":           true,
		"s.required_ruby_version":         true,
		"spec.required_ruby_version":      true,
	}
	rootPath      *string
	dependencyMap = make(map[string]VersionMap)
)

const (
	Title                 = "specs:"
	specDependencyPath    = "vendor/bundle/ruby"
	specExtension         = ".gemspec"
	specDefaultDir        = "specifications"
	cacheDefaultDir       = "cache"
	gemDefaultDir         = "gems"
	rakeFileDefaultName   = "Rakefile"
	platformsDefaultName  = "PLATFORMS"
	legacyLockExtension   = ".lock"
	licenseDefaultFile    = "LICENSE"
	copyrightDefaultLabel = "Copyright (c)"
	lockExtension         = ".locked"
	gemDefaultExtension   = ".gem"
	detectionModeSpec     = "spec"
	detectionModeLock     = "lock"
	noAssertion           = "NO ASSERTION"
)

type (
	Line struct {
		Position  int
		Value     string
		Relations []string
	}
	Package struct {
		Name      string
		Relations []string
	}
	Spec struct {
		Name                    string
		License                 string
		Licenses                []string
		LicenseText             string
		GemLocationDir          string
		CopyRight               string
		Version                 string
		Emails                  []string
		HomePage                string
		Authors                 []string
		Summary                 string
		Description             string
		Checksum                string
		RuntimeDependencies     []string
		DevelopmentDependencies []string
		RubyGemsVersion         string
		RequiredRubyVersion     string
		Specifications          []Spec
	}
	VersionMap struct {
		versions map[string]Spec
		count    int
	}
	DependencyMap map[string]VersionMap
)

// Returns the root module
func getGemRootModule(path string) (*meta.Package, error) {
	paths, secPaths, err := getGemPaths()
	if err != nil {
		return nil, fmt.Errorf("getting gem paths: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		initializeDepCache(&wg, paths, secPaths)
	}()
	wg.Wait()

	rootPath = &path
	rootModule := meta.Package{}
	rootModule.Packages = make(map[string]*meta.Package)
	spec, err := getSpecDependencies(path)
	if err != nil {
		return nil, err
	}
	var supplier meta.Supplier
	authors := spec.Authors
	if len(authors) > 0 {
		supplier.Type = meta.Person
		supplier.Name = authors[0]
	}

	setLicenseInfo(spec.GemLocationDir, &rootModule)
	rootModule.Name = gemName(spec.Name)
	rootModule.Version = spec.Version
	rootModule.Supplier = supplier
	rootModule.Root = true
	rootModule.Path = spec.GemLocationDir
	rootModule.PackageHomePage = cleanURI(spec.HomePage)
	rootModule.PackageDownloadLocation = cleanURI(spec.HomePage)
	rootModule.PackageURL = cleanURI(spec.HomePage)
	rootModule.Checksum = meta.Checksum{
		Algorithm: meta.HashAlgoSHA256,
		Value:     spec.Checksum,
	}

	return &rootModule, nil
}

// Returns the root module and associated dependencies
func listGemRootModule(path string) ([]meta.Package, error) {
	rootPath = &path
	modules := make([]meta.Package, 0)
	noSpecs := make(map[string]bool)

	layerOneGems,
		layerTwoGems,
		layerThreeGems :=
		make([]meta.Package, 0),
		make([]meta.Package, 0),
		make([]meta.Package, 0)

	_1stLayerMapped, _2ndLayerMapped, _3rdLayerMapped :=
		make(map[string]bool),
		make(map[string]bool),
		make(map[string]bool)

	var firstLayerModule,
		secondLayerModule meta.Package

	// Parent Layer - Root
	rootModule, err := getGemRootModule(path)
	if err != nil {
		return nil, err
	}

	modules = append(modules, *rootModule)
	rootSpec, err := getSpecDependencies(path)
	if err != nil {
		return nil, err
	}

	// Populate Child Layers
	for _, dep := range rootSpec.Specifications {

		// Don't include root in children
		if dep.Name == rootModule.Name && dep.Version == rootModule.Version {
			continue
		}

		parentLayerModule := parseSpec(dep)
		setLicenseInfo(dep.GemLocationDir, &parentLayerModule)

		for _, firstDescendant := range dep.RuntimeDependencies {

			firstDescendantSpec, name, err := getDescendantInfo(firstDescendant)
			if err != nil {
				l1 := fmt.Sprintf("%s runtime dependency of %s", firstDescendant, dep.Name)
				noSpecs[l1] = true
				continue
			}
			// Add 1st Layer
			layerOneGems, firstLayerModule = addGemLayer(firstDescendantSpec, name, &parentLayerModule, _1stLayerMapped, layerOneGems)

			for _, secondDescendant := range firstDescendantSpec.RuntimeDependencies {
				secondDescendantSpec, name, err := getDescendantInfo(secondDescendant)
				if err != nil {
					l2 := fmt.Sprintf("%s runtime dependency of %s", secondDescendant, firstDescendantSpec.Name)
					noSpecs[l2] = true
					continue
				}
				// Add 2nd Layer
				layerTwoGems, secondLayerModule = addGemLayer(secondDescendantSpec, name, &firstLayerModule, _2ndLayerMapped, layerTwoGems)

				for _, thirdDescendant := range secondDescendantSpec.RuntimeDependencies {
					thirdDescendantSpec, name, err := getDescendantInfo(thirdDescendant)
					if err != nil {
						l3 := fmt.Sprintf("%s runtime dependency of %s", thirdDescendant, secondDescendantSpec.Name)
						noSpecs[l3] = true
						continue
					}
					// Add 3rd Layer
					layerThreeGems, _ = addGemLayer(thirdDescendantSpec, name, &secondLayerModule, _3rdLayerMapped, layerThreeGems)
				}

			}

		}

		rootModule.Packages[dep.Name] = &parentLayerModule
		modules = append(modules, parentLayerModule)

	}

	// Adds related dependencies
	modules = append(modules, layerOneGems...)
	modules = append(modules, layerTwoGems...)
	modules = append(modules, layerThreeGems...)

	if len(noSpecs) > 0 {
		for dep := range noSpecs {
			log.Warnf("manifest for %s not found in gem paths", dep)
		}
	}

	return modules, nil
}

// Parses spec info into module object
func parseSpec(spec Spec) meta.Package {

	var supplier meta.Supplier
	authors := spec.Authors
	if len(authors) > 0 {
		supplier.Type = meta.Person
		supplier.Name = authors[0]
	}
	return meta.Package{
		Name:                    gemName(spec.Name),
		Version:                 spec.Version,
		Root:                    false,
		PackageHomePage:         cleanURI(spec.HomePage),
		PackageDownloadLocation: cleanURI(spec.HomePage),
		Supplier:                supplier,
		PackageURL:              cleanURI(spec.HomePage),
		Checksum: meta.Checksum{
			Algorithm: meta.HashAlgoSHA256,
			Value:     spec.Checksum,
		},
		Packages: make(map[string]*meta.Package),
	}

}

// Adds a new layer to the dependency tree
func addGemLayer(descendant Spec, name string, parent *meta.Package, layer map[string]bool, gems []meta.Package) ([]meta.Package, meta.Package) {
	descendantModule := parseSpec(descendant)
	setLicenseInfo(descendant.GemLocationDir, &descendantModule)
	return setChildModule(name, parent, &descendantModule, layer, gems), descendantModule
}

// Sets the child of a parent module
func setChildModule(name string, parent, child *meta.Package, layer map[string]bool, gems []meta.Package) []meta.Package {
	parent.Packages[name] = child
	if !layer[child.Name] {
		gems = append(gems, *child)
		layer[child.Name] = true
	}
	return gems
}

// Sets license info from generic helper
func setLicenseInfo(path string, module *meta.Package) {

	licensePkg, err := helper.GetLicenses(path)
	if err == nil {
		module.LicenseDeclared = helper.BuildLicenseDeclared(licensePkg.ID)
		module.LicenseConcluded = helper.BuildLicenseConcluded(licensePkg.ID)
		if !helper.LicenseSPDXExists(licensePkg.ID) {
			licensePkg.ID = fmt.Sprintf("LicenseRef-%s", licensePkg.ID)
			module.OtherLicense = append(module.OtherLicense, *licensePkg)
		}
		module.Copyright = helper.GetCopyright(licensePkg.ExtractedText)
		module.CommentsLicense = licensePkg.Comments
		module.LocalPath = path
	}

}

// Gets gem info from in-memory cache
func getDescendantInfo(child string) (Spec, string, error) {
	name, _ := childDepInfo(child)
	version := getRangedVersion(child)
	info := lookupGemInfo(strings.TrimSpace(name), strings.TrimSpace(version))
	if info.Name == "" {
		return Spec{}, "", errors.New("not found")
	}
	return info, name, nil
}

// Gets parent and child dependency tree from .gemspec
func getSpecDependencies(path string) (Spec, error) {

	manifest, err := detectManifest(path, detectionModeSpec)
	if err != nil {
		return Spec{}, err
	}
	module := getSpecs(filepath.Join(path, manifest))
	BuildSpecDependencies(filepath.Join(path, specDependencyPath), false, &module)
	return module, nil
}

// Gets parent and child dependency tree from Gemfile.lock
func GetLockedDependencies(path string) ([]Package, error) {

	manifest, err := detectManifest(path, detectionModeLock)
	if err != nil {
		return []Package{}, err
	}
	BuildLockDependencyTree(Content(filepath.Join(path, manifest)))
	if hasNodes(dependencies) {
		return dependencies, nil
	}
	return nil, errors.New("no dependencies were found for project")
}

// Builds parent and child dependency tree from .gemspec
func BuildSpecDependencies(path string, isFullPath bool, module *Spec) {

	files, err := os.ReadDir(path)

	if err != nil {
		log.Fatal(err)
	}

	if !isFullPath {
		for _, dir := range files {
			if dir.IsDir() {
				fullPath := filepath.Join(path, dir.Name(), specDefaultDir)
				BuildSpecDependencies(fullPath, true, module)
				return
			}
		}
	}

	cachePath := strings.Replace(path, specDefaultDir, cacheDefaultDir, 1)

	name, version, _ := rootGem(cachePath, cleanName(module.Name))
	versionedName := fmt.Sprintf("%s-%s", name, version)

	rootSha, err := checkSum(cachePath, versionedName)
	if err == nil && rootSha != "" {
		module.Checksum = rootSha
	}
	if module.Checksum == "" {
		module.Checksum = noAssertion
	}

	copyRight, LicenseText, LicensePath, err := extractRootLicense(*rootPath)
	if err == nil {
		module.CopyRight = copyRight
		module.LicenseText = LicenseText
		module.GemLocationDir = LicensePath
	}

	for i, f := range files {
		if filepath.Ext(f.Name()) == specExtension {

			specPath := filepath.Join(path, f.Name())

			module.Specifications = append(module.Specifications, getSpecs(specPath))
			fileName := cleanName(strings.Replace(f.Name(), specExtension, "", 1))

			sha, err := checkSum(cachePath, fileName)
			if err == nil {
				module.Specifications[i].Checksum = sha
			}
			if module.Specifications[i].Checksum == "" {
				module.Specifications[i].Checksum = noAssertion
			}
			module.Specifications[i].Name = fileName

			copyRight, LicenseText, LicensePath, err := extractLicense(specDependencyPath, fileName, false)
			if err == nil {
				module.Specifications[i].CopyRight = copyRight
				module.Specifications[i].LicenseText = LicenseText
				module.Specifications[i].GemLocationDir = LicensePath
				module.Specifications[i].Version = gemVersion(fileName)
			} else {
				log.Error(err)
			}

		}
	}

}

// launches routines to Get metadata from .gemspec concurrently
func getSpecs(path string) Spec {

	output := make(chan Spec, 1)
	go mapSpec(Content(path), output)
	return <-output
}

// Speeds up the build process concurrently
func mapSpec(rows []string, output chan<- Spec) {

	spec := Spec{}
	for _, row := range rows {
		column := columns(row)
		if !Required[column] && invalidRow(row) {
			continue
		}
		ReduceSpec(row, column, &spec)
	}
	output <- spec
}

// Speeds up the build process concurrently
func ReduceSpec(row, column string, spec *Spec) {

	switch strings.Trim(column, " ") {
	case "s.name":
		fallthrough
	case "spec.name":
		_, value := strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], " ", ""), strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
		spec.Name = cleanName(unfreeze(value))
	case "s.license":
		fallthrough
	case "spec.license":
		_, value := strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], " ", ""), strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
		spec.License = value
	case "s.email":
		fallthrough
	case "spec.email":
		_, value := strings.SplitN(strings.TrimLeft(row, "="), " ", 2)[0], strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
		spec.Emails = list(value)
	case "s.licenses":
		fallthrough
	case "spec.licenses":
		_, value := strings.SplitN(strings.TrimLeft(row, "="), " ", 2)[0], strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
		spec.Licenses = list(value)
	case "s.homepage":
		fallthrough
	case "spec.homepage":
		_, value := strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], " ", ""), strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
		spec.HomePage = cleanURI(unfreeze(value))
	case "s.authors":
		fallthrough
	case "spec.authors":
		spec.Authors = getAuthors(row)
	case "s.summary":
		fallthrough
	case "spec.summary":
		_, value := strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], " ", ""), strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1]
		spec.Summary = unfreeze(value)
	case "s.description":
		fallthrough
	case "spec.description":
		_, value := strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], " ", ""), strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1]
		spec.Description = unfreeze(value)
	case "s.rubygems_version":
		fallthrough
	case "spec.rubygems_version":
		_, value := strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], " ", ""), strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
		spec.RubyGemsVersion = unfreeze(value)
	case "s.required_ruby_version":
		fallthrough
	case "spec.required_ruby_version":
		_, value := strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], " ", ""), strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
		spec.RequiredRubyVersion = unfreeze(value)
	case "s.add_runtime_dependency":
		fallthrough
	case "spec.add_runtime_dependency":
		if strings.ContainsAny(row, "[]") {
			value := fmt.Sprintf("%s%s%s", clean(row, "<", ">"), " ", clean(row, "[", "]"))
			spec.RuntimeDependencies = append(spec.RuntimeDependencies, value)
		} else {
			_, value := strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
			spec.RuntimeDependencies = append(spec.RuntimeDependencies, value)
		}
	case "s.add_dependency":
		fallthrough
	case "spec.add_dependency":
		if strings.ContainsAny(row, "[]") {
			value := fmt.Sprintf("%s%s%s", clean(row, "<", ">"), " ", clean(row, "[", "]"))
			if !isDuplicate(value, *spec) {
				spec.RuntimeDependencies = append(spec.RuntimeDependencies, value)
			}

		} else {
			_, value := strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
			if !isDuplicate(value, *spec) {
				spec.RuntimeDependencies = append(spec.RuntimeDependencies, value)
			}
		}
	case "s.add_development_dependency":
		fallthrough
	case "spec.add_development_dependency":
		if strings.ContainsAny(row, "[]") {
			value := fmt.Sprintf("%s%s%s", clean(row, "<", ">"), " ", clean(row, "[", "]"))
			if !isDuplicate(value, *spec) {
				spec.DevelopmentDependencies = append(spec.DevelopmentDependencies, value)
			}
		} else {
			_, value := strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[0], strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), " ", 2)[1], " ", "")
			if !isDuplicate(value, *spec) {
				spec.DevelopmentDependencies = append(spec.DevelopmentDependencies, value)
			}
		}

	}

}

// Checks for duplicate entries
func isDuplicate(value string, spec Spec) bool {

	var skip bool
	if len(spec.RuntimeDependencies) == 0 {
		return skip
	}
	for _, dep := range spec.RuntimeDependencies {
		if dep == value {
			skip = true
			break
		}
	}
	return skip
}

// Builds Dependency Tree from Gemfile.lock
func BuildLockDependencyTree(rows []string) {

	var startIndex, specPosition int
	var newLines []string

	for i, line := range rows {

		value := strings.Trim(line, " ")

		currentPosition := len(line) - len(strings.TrimLeft(line, " "))

		if value == Title {
			startIndex = i
			specPosition = currentPosition
		}

		if startIndex > 0 {
			lines = append(lines, Line{
				Position: currentPosition,
				Value:    value,
			})
			newLines = append(newLines, value)
		}

		if currentPosition < specPosition {

			linesToRead := lines[indexOf(Title, newLines)+1:]

			if hasLines(linesToRead) {
				buildTree(linesToRead)
				newLines = []string{}
				lines = []Line{}

			} else {
				break
			}
		}

	}
}

// Compute SHA 256 Checksum for gems
func checkSum(path string, filename string) (sha string, err error) {
	// I don't get this. Is it an error?
	if !strings.Contains(path, cacheDefaultDir) {
		return "", nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path, err = gemDir()
		if err != nil {
			return "", fmt.Errorf("getting gemdir: %w", err)
		}
	}

	// This usedto have per arch shell outs to
	// sha256 and others, i swapped for the internal
	// pure go implementation.
	sha256, err := getSHA(filepath.Join(path, filename+gemDefaultExtension))
	if err != nil {
		return "", err
	}
	return sha256, nil
}

// Gets the root dependency name and version
func rootGem(path string, filename string) (string, string, error) {
	var name, version string
	files, err := os.ReadDir(path)
	if err != nil {
		return "", "", fmt.Errorf("reading path: %w", err)
	}
	for _, f := range files {

		if strings.LastIndex(f.Name(), "-") == -1 {
			continue
		}
		stp := strings.LastIndex(f.Name(), "-")
		runes := []rune(f.Name())
		n := string(runes[0:stp])

		v := strings.Replace(string(runes[stp+1:]), ".gem", "", 1)

		filename = strings.ReplaceAll(filename, `="`, "")
		filename = strings.ReplaceAll(filename, `"`, "")

		name = n

		if name == filename {
			version = v
			break
		}
	}

	return name, version, nil
}

// Extracts Root License Info
func extractRootLicense(path string) (string, string, string, error) {
	var copyright string
	var text string
	var licensePath string

	files, err := os.ReadDir(path)
	if err != nil {
		return "", "", "", fmt.Errorf("reading dir %s: %w", path, err)
	}
	licensePath = path
	for _, f := range files {
		if strings.Contains(f.Name(), licenseDefaultFile) {
			path = filepath.Join(path, f.Name())
			break
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", fmt.Errorf("reading file: %w", err)
	}
	text = string(data)
	rows := Content(path)
	for _, row := range rows {
		if strings.Contains(row, copyrightDefaultLabel) {
			copyright = row
			break
		}
	}

	return copyright, text, licensePath, nil

}

// Extracts Child License Info
func extractLicense(path string, filename string, isFullPath bool) (string, string, string, error) {
	var copyright string
	var text string
	var licensePath string

	if !isFullPath {
		files, err := os.ReadDir(path)
		if err != nil {
			return "", "", "", err
		}

		for _, f := range files {
			if f.IsDir() {
				fullPath := filepath.Join(path, f.Name())
				return extractLicense(fullPath, filename, true)
			}
		}
	} else {
		if !strings.Contains(path, gemDefaultDir) {
			path = filepath.Join(path, gemDefaultDir, filename)
		}
		files, err := os.ReadDir(path)
		if err != nil {
			return "", "", "", fmt.Errorf("reading directory: %w", err)
		}
		licensePath = path
		sampleLicenses := []string{licenseDefaultFile, "GPL", "LGPL", "PSFL", "LICENCE.txt"}
		for _, f := range files {
			if strings.ContainsAny(f.Name(), strings.Join(sampleLicenses, " ")) {
				path = filepath.Join(path, f.Name())
				break
			}
		}
		data, err := os.ReadFile(path)
		if err == nil {
			text = string(data)
		}
		rows := Content(path)
		for _, row := range rows {
			if strings.Contains(row, copyrightDefaultLabel) {
				copyright = row
				break
			}
		}

	}
	return copyright, text, licensePath, nil

}
func validateProjectType(path string) bool {
	if _, err := detectManifest(path, detectionModeSpec); err != nil {
		return false
	}
	return true
}

// Constructs dependency tree recursively
func buildTree(linesToRead []Line) {
	var startIndex, stopIndex int
	var children []Line
	var nextBatch = []Line{}

	lastPosition := linesToRead[0].Position

	for i, line := range linesToRead {

		if line.Position > lastPosition {
			if startIndex == 0 {
				startIndex = i
			}
		}
		if line.Position == lastPosition && i > 0 {
			if stopIndex == 0 {
				stopIndex = i
				break
			}
		}
	}

	if stopIndex > 0 {
		children = linesToRead[startIndex:stopIndex]

	} else {
		children = linesToRead[startIndex:]
	}

	parent := linesToRead[0].Value

	dependency := Package{Name: parent}

	for _, line := range children {
		if line.Value == parent {
			continue
		}
		dependency.Relations = append(dependency.Relations, line.Value)
	}

	dependencies = append(dependencies, dependency)

	if stopIndex > 0 {
		nextBatch = linesToRead[stopIndex:]
	}

	if hasLines(nextBatch) {
		buildTree(nextBatch)
	}
}

// childDepInfo reads child dependency info
func childDepInfo(value string) (name, version string) {
	stp := strings.Index(value, `"`)
	runes := []rune(value)
	name = string(runes[0:stp])
	rn := strings.Split(string(runes[stp:]), ",")
	for i := 0; i < len(rn); i++ {
		version += strings.ReplaceAll(strings.ReplaceAll(rn[i], " ", ""), `"`, "")
	}
	if strings.ContainsAny(version, "#{") {
		version = noAssertion
	}

	return name, version
}

// Scans the provided path for ecosystem manifest file
func detectManifest(path, mode string) (string, error) {

	var manifest string
	var err error
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {

		switch mode {
		case detectionModeLock:
			if filepath.Ext(f.Name()) == lockExtension || filepath.Ext(f.Name()) == legacyLockExtension {
				manifest = f.Name()
				err = errors.New("No file with extension '.lock' was detected in " + path)
			}
		case detectionModeSpec:
			if filepath.Ext(f.Name()) == specExtension {
				manifest = f.Name()
				err = errors.New("No file with extension '.gemspec' was detected in " + path)
			}
		}

		if manifest != "" {
			err = nil
			break
		}

	}
	if manifest == "" {
		err = errors.New("gemspec file not found in projects root directory")
		return manifest, err
	}
	return manifest, err
}

// Wow this is actually modifying the project :/
// Detect whether current OS is added in the Gemfile.lock PLATFORMS section
// Add if not detected for better user experience
func ensurePlatform(path string) bool {
	manifest, err := detectManifest(path, detectionModeLock)
	if err != nil || manifest == "" {
		return false
	}
	path = fmt.Sprintf("%s%s", path, manifest)
	beginChar := path[0:1]

	followedByChar := path[1:2]
	if beginChar == "." && followedByChar != "/" {
		path = strings.Replace(path, ".", "./", 1)
	}

	lines := Content(path)

	fileContent := ""
	index, indent := getInsertIndex(lines)
	space := ""
	for i := 0; i < indent; i++ {
		space += " "
	}
	str := fmt.Sprintf("%s%s\n", space, runtime.GOOS)
	for i, line := range lines {
		if strings.TrimLeft(line, " ") == runtime.GOOS {
			return false
		}
		if i == index {
			fileContent += str
		}
		fileContent += line
		fileContent += "\n"
	}
	return os.WriteFile(path, []byte(fileContent), 0644) == nil // nolint:gosec
}

// Get exact index in file to append current OS value
func getInsertIndex(rows []string) (int, int) {

	var index, position, currentPosition int
	var PlatformFound bool
	for i, line := range rows {
		value := strings.Trim(line, " ")
		currentPosition = len(line) - len(strings.TrimLeft(line, " "))
		if value == platformsDefaultName {
			PlatformFound = true
		}
		if PlatformFound && value == "" {
			index = i
			break
		}
		position = currentPosition
	}
	return index, position
}

// Get local gem paths from env
func getGemPaths() ([]string, []string, error) {
	var start, stop, reading bool
	locations, secondaryLocation := []string{}, []string{}
	cmd := exec.Command("gem", "env")
	output, err := cmd.Output()
	if err != nil {
		return locations, secondaryLocation, fmt.Errorf("getting env from gem: %w", err)
	}
	paths := strings.Fields(string(output))
	for i, path := range paths {
		start = paths[i] == "GEM" && paths[i+1] == "PATHS:"
		stop = paths[i] == "GEM" && paths[i+1] == "CONFIGURATION:"
		if stop {
			break
		}
		if start {
			reading = true
			if strings.Contains(path, gemDefaultExtension) {
				secondaryLocation = append(secondaryLocation, path)
			} else if path != "GEM" && path != "PATHS:" {
				locations = append(locations, path)
			}

		}
		if reading && path != "-" && path != "GEM" && path != "PATHS:" {
			if strings.Contains(path, gemDefaultExtension) {
				secondaryLocation = append(secondaryLocation, path)
			} else {
				locations = append(locations, path)
			}
		}
	}
	return locations, secondaryLocation, nil
}

// Build tree mapping from all gems detected in gem paths
func buildLocalTree(paths []string, secondaryLocation string) []Spec {

	localSpecs := []Spec{}
	primaryLocation, _ := gemDir()

	for _, installPath := range paths {
		specPath := filepath.Join(installPath, specDefaultDir)
		cachePath := filepath.Join(installPath, cacheDefaultDir)
		checkSumPaths := []string{cachePath, secondaryLocation, primaryLocation}
		licensePath := filepath.Join(installPath, gemDefaultDir)

		if _, err := os.Stat(specPath); err != nil {
			continue
		}

		files, err := os.ReadDir(specPath)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range files {
			if !f.IsDir() && strings.Contains(f.Name(), specExtension) {
				fullSpecsPath := filepath.Join(specPath, f.Name())
				spec := getSpecs(fullSpecsPath)
				if spec.Version == "" {
					spec.Version = getExistingVersion(cleanName(spec.Name))
				}
				fileName := strings.Replace(f.Name(), specExtension, "", 1)

				for _, csp := range checkSumPaths {
					if _, err := os.Stat(csp); os.IsNotExist(err) {
						continue
					}
					sha, err := checkSum(csp, fileName)
					if err == nil && sha != "" {
						spec.Checksum = sha
						break
					}
				}
				if spec.Checksum == "" {
					spec.Checksum = noAssertion
				}
				fullLicensePath := filepath.Join(licensePath, fileName)
				copyRight, LicenseText, LicensePath, err := extractLicense(fullLicensePath, fileName, true)
				if err == nil {
					spec.CopyRight = copyRight
					spec.LicenseText = LicenseText
					spec.GemLocationDir = LicensePath
				}
				localSpecs = append(localSpecs, spec)
			}
		}
	}

	return localSpecs
}

// Selects gem info from existing versions in cache
func lookupGemInfo(name, version string) Spec {

	versionedSpecs := []Spec{}
	versions := dependencyMap[strings.TrimSpace(name)].versions
	latestVersionInfo := Spec{}
	if len(versions) == 0 {
		return Spec{}
	}
	if versions[version].Version != "" {
		return versions[version]
	}
	for _, spec := range versions {
		versionedSpecs = append(versionedSpecs, spec)
	}
	if len(versionedSpecs) == 1 {
		return versionedSpecs[0]
	}
	latestVersionInfo = versionedSpecs[0]

	for _, spec := range versions {
		if currentVersion, err := strconv.Atoi(strings.Split(spec.Version, ".")[0]); err == nil {
			if latestVersion, err := strconv.Atoi(strings.Split(latestVersionInfo.Version, ".")[0]); err == nil && currentVersion > latestVersion {
				latestVersionInfo = spec
			}
		}
	}

	return latestVersionInfo
}

// Initialize in-memory dependency cache
func initializeDepCache(wg *sync.WaitGroup, paths, secPaths []string) {
	secondaryCachePath, _ := gemDir()
	if len(secPaths) > 0 {
		secondaryCachePath = filepath.Join(secPaths[0], cacheDefaultDir)
	}
	depSpecs := buildLocalTree(paths, secondaryCachePath)
	for _, dep := range depSpecs {
		name, v := cleanName(dep.Name), dep.Version
		if dependencyMap[name].count > 0 {
			tempVersion := dependencyMap[name]
			dep.Name = cleanName(dep.Name)
			tempVersion.versions[v] = dep
			tempVersion.count = len(tempVersion.versions)
			dependencyMap[name] = tempVersion
		} else {
			dependencyMap[name] = VersionMap{
				versions: make(map[string]Spec),
				count:    1,
			}
			dep.Name = cleanName(dep.Name)
			dependencyMap[name].versions[v] = dep
		}
	}
	wg.Done()
}

// gets version from specified range
func getRangedVersion(rv string) string {
	_, v := childDepInfo(rv)
	if strings.ContainsAny(v, "~>") {
		v = strings.Fields(strings.ReplaceAll(v, "~>", " "))[0]
		if strings.Contains(v, ">") {
			v = strings.Fields(strings.ReplaceAll(v, ">", " "))[0]
		}
	}
	return v
}

// gets version existing on file system
func getExistingVersion(gem string) string {

	cmd := exec.Command("gem", "query", "-e", gem)
	output, err := cmd.Output()
	if err != nil {
		return noAssertion
	}
	lines := strings.Fields(string(output))
	var s string
	for i, row := range lines {
		for _, c := range row {
			val := string(c)
			if _, err := strconv.Atoi(val); err == nil || val == "." {
				s += val
			}
		}
		if i > 0 {
			s += " "
		}

	}
	if s == "" {
		return ""
	}
	return strings.Fields(s)[0]
}

// Scans and return file content
func Content(path string) []string {
	file, err := os.Open(path)
	record := []string{}
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		record = append(record, line)
	}
	if err := scanner.Err(); err != nil {
		return record
	}
	return record
}

// Get the first column of a row
func columns(row string) string {
	var separator = " "
	if strings.Contains(row, "(") {
		separator = "("
	}
	return strings.SplitN(strings.TrimLeft(row, " "), separator, 2)[0]
}

// Build a slice from a row 'email,author ...'
func list(row string) []string {

	val := strings.ReplaceAll(strings.SplitN(strings.TrimLeft(row, " "), "=", 2)[1], "[", "")
	val = strings.ReplaceAll(val, "]", "")
	val = unfreeze(val)
	return strings.Split(val, ",")
}

// Track element positions
func indexOf(element string, data []string) int {

	for k, v := range data {
		if strings.Trim(v, " ") == element {
			return k
		}
	}
	return -1
}

// Validate data rows
func invalidRow(row string) bool {

	return !strings.Contains(strings.ReplaceAll(row, " ", ""), "s.add_development_dependency") ||
		!strings.Contains(strings.ReplaceAll(row, " ", ""), "spec.add_development_dependency") &&
			!strings.Contains(strings.ReplaceAll(row, " ", ""), "s.add_dependency") ||
		!strings.Contains(strings.ReplaceAll(row, " ", ""), "spec.add_dependency") ||
		strings.ContainsAny(row, "#{")
}

// Remove unwanted symbols & characters if exists
func clean(val, from, to string) string {
	a := strings.SplitN(val, from, 2)[1]
	a = strings.SplitN(a, to, 2)[0]
	return a
}

// Sanitize names from unknown chars
func cleanName(name string) string {

	s := strings.ReplaceAll(name, "=", "")
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.ReplaceAll(s, "“", "")
	s = strings.ReplaceAll(s, "'", "")
	return s
}

// Sanitize URI
func cleanURI(url string) string {
	u := strings.ReplaceAll(url, "=", "")
	u = strings.ReplaceAll(u, "\"", "")
	return u
}

// Get package version
func gemVersion(name string) string {

	if !strings.Contains(name, "-") {
		return ""
	}
	stp := strings.LastIndex(name, "-")
	runes := []rune(name)
	return string(runes[stp+1:])
}

// Get name without the version
func gemName(name string) string {
	if !strings.Contains(name, "-") {
		return name
	}
	stp := strings.LastIndex(name, "-")
	vRunes := []rune(name)
	v := string(vRunes[stp+1:])
	if len(v) < 1 {
		return name[:stp]
	}
	c := v[:1]
	if _, err := strconv.ParseInt(c, 10, 32); err != nil {
		return name
	}
	runes := []rune(name)
	return string(runes[:stp])
}

// Remove unwanted word if exists '.freeze is often added by bundler'
func unfreeze(val string) string {

	return strings.ReplaceAll(val, ".freeze", "")
}

// Check if package slice contains elements
func hasNodes(object []Package) bool {

	return len(object) > 0
}

// Check if line slice contains elements
func hasLines(object []Line) bool {

	return len(object) > 0
}

// Auto create Rakefile if not detected
func hasRakefile(path string) bool {
	filename := filepath.Join(path, rakeFileDefaultName)
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return os.WriteFile(filename, []byte("require \"bundler/gem_tasks\" \ntask :default => :spec"), 0600) == nil
}

// Gets the gem installation directory
func gemDir() (string, error) {
	cmd := exec.Command("gem", "environment", "gemdir")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting gemdir from env: %w", err)
	}
	return filepath.Join(strings.Fields(string(output))[0], cacheDefaultDir), nil
}

// extracts authors from row
func getAuthors(row string) []string {
	value := strings.SplitN(strings.TrimLeft(unfreeze(row), " "), " ", 2)[1]
	s := []string{`[`, `]`, `"`, `=`}
	for _, v := range s {
		value = strings.ReplaceAll(value, v, "")
	}
	return strings.Split(value, ",")
}

// computes the SHA 256 checkSum of a gem
func getSHA(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("writing to hash: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
