package npm

import (
	"testing"

	"github.com/opensbom-generator/parsers/meta"
	"github.com/stretchr/testify/assert"
)

func TestParseIntegrityV2(t *testing.T) {

	cs, err := ParseIntegrityV2("sha512-TIGnTpdo+E3+pCyAluZvtED5p5wCqLdezCyhPZzKPcxvFplEt4i+W7OONCKgeZFT3+y5NZZfOOS/Bdcanm1MYA==")
	assert.Equal(t, cs.Algorithm, meta.HashAlgoSHA512)
	assert.Equal(t, cs.Value, "4c81a74e9768f84dfea42c8096e66fb440f9a79c02a8b75ecc2ca13d9cca3dcc6f169944b788be5bb38e3422a0799153dfecb935965f38e4bf05d71a9e6d4c60")
	assert.Equal(t, err, nil)
}

func TestPackageV2ToMeta(t *testing.T) {
	// parse existing package lock v2 file
	data, _ := ReadManifest("testdata/package-lock-v2.json")
	lock, _ := ParseManifestV2(data)
	pkg, err := PackageV2ToMeta("node_modules/ansi-regex", lock.Packages["node_modules/ansi-regex"])
	assert.Nil(t, err)
	assert.Equal(t, pkg.Name, "node_modules/ansi-regex")
	assert.Equal(t, pkg.Version, "2.1.1")
	assert.Equal(t, pkg.Supplier.Name, "")
	assert.Equal(t, pkg.Checksum.Algorithm, meta.HashAlgoSHA512)
	assert.Equal(t, pkg.PackageDownloadLocation, "https://registry.npmjs.org/ansi-regex/-/ansi-regex-2.1.1.tgz")
}
