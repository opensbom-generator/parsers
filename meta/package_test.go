// SPDX-License-Identifier: Apache-2.0
// Copyright 2023 The Linux Foundation and its contributors

package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHashAlgorithm(t *testing.T) {
	algo := GetHashAlgorithm("sha256")
	assert.Equal(t, algo, HashAlgoSHA256)
	algo = GetHashAlgorithm("mD5")
	assert.Equal(t, algo, HashAlgoMD5)
	algo = GetHashAlgorithm("base64")
	assert.Equal(t, algo, HashAlgoUnsupported)
}
