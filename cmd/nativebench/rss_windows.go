//go:build windows

package main

import "os"

// maxRSS is not measured on Windows: the pre-registration is macOS arm64 only.
func maxRSS(*os.ProcessState) int64 { return 0 }
