//go:build !windows

package logger

import "io"

func enableANSI(io.Writer) {}
