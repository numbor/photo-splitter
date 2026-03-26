//go:build !windows

package scan

import "errors"

func AcquireScanJPEG(outputPath string) error {
	_ = outputPath
	return errors.New("la scansione WIA è supportata solo su Windows")
}

func AcquireScanJPEGWithOptions(outputPath string, options Options) error {
	_ = outputPath
	_ = options
	return errors.New("la scansione WIA è supportata solo su Windows")
}

func AcquireScanTIFFWithOptions(outputPath string, options Options) error {
	_ = outputPath
	_ = options
	return errors.New("la scansione WIA è supportata solo su Windows")
}
