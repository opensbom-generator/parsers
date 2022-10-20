// SPDX-License-Identifier: Apache-2.0

package models

// OutputFormat defines an int enum of supported output formats
type OutputFormat int

const (
	OutputFormatSpdx OutputFormat = iota
	OutputFormatJson
)
