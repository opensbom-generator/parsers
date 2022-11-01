// SPDX-License-Identifier: Apache-2.0

package javagradle

import (
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/opensbom-generator/parsers/internal/helper"
)

// use newGradleExec to instantiate
type gradleExec struct {
	executable string
	workingDir string
}

func newGradleExec(workingDir string) gradleExec {
	ge := gradleExec{}

	if hasGradlew(workingDir) {
		ge.executable = "./gradlew"
	} else {
		ge.executable = "gradle"
	}
	ge.workingDir = workingDir
	return ge
}

func hasGradlew(workingDir string) bool {
	return helper.Exists(filepath.Join(workingDir, "gradlew")) || (runtime.GOOS == "windows" && helper.Exists(filepath.Join(workingDir, "gradlew.bat")))
}

func (ge gradleExec) run(args ...string) *exec.Cmd {
	args = append(args, "--console=plain")
	// TODO Use internal helper, migrate to k8s command
	cmd := exec.Command(ge.executable, args...) // nolint :gosec:This is fixed to the gradle command
	cmd.Dir = ge.workingDir
	return cmd
}
