package main

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./gl PATTERN dir")
		return
	}

	pattern := os.Args[1]
	dir := os.Args[2]

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			switch {
			case strings.HasSuffix(path, ".zip"), strings.HasSuffix(path, ".gz"), strings.HasSuffix(path, ".tgz"), strings.HasSuffix(path, ".zstd"):
				handleCompressedFile(path, pattern)
			default:
				runAg(pattern, path)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %q: %v\n", dir, err)
	}

	cleanupLogs(dir)
}

func runAg(pattern, path string) {
	cmd := exec.Command("ag", "--color", pattern, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running ag on %s: %s\n", path, err)
		return
	}
	fmt.Printf("Results for %s:\n%s\n", path, output)
}

func handleCompressedFile(path, pattern string) {
	fmt.Printf("Handling compressed file: %s\n", path)
	ext := filepath.Ext(path)

	var reader io.ReadCloser
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening file %s: %s\n", path, err)
		return
	}
	defer file.Close()

	switch ext {
	case ".gz", ".tgz":
		if reader, err = gzip.NewReader(file); err != nil {
			fmt.Printf("Error creating gzip reader for %s: %s\n", path, err)
			return
		}
	case ".zip":
		fileInfo, err := file.Stat()
		if err != nil {
			fmt.Printf("Error getting file info for %s: %s\n", path, err)
			return
		}
		zr, err := zip.NewReader(file, fileInfo.Size())
		if err != nil {
			fmt.Printf("Error reading zip file %s: %s\n", path, err)
			return
		}
		for _, f := range zr.File {
			if reader, err = f.Open(); err != nil {
				fmt.Printf("Error opening zip file content %s: %s\n", f.Name, err)
				return
			}
			runAgThroughReader(pattern, reader, f.Name)
			reader.Close()
		}
		return
	case ".zstd":
		decoder, err := zstd.NewReader(file)
		if err != nil {
			fmt.Printf("Error creating zstd decoder for %s: %s\n", path, err)
			return
		}
		defer decoder.Close()
		reader = decoder.IOReadCloser()
	default:
		fmt.Printf("Unsupported file extension: %s\n", ext)
		return
	}

	runAgThroughReader(pattern, reader, path)
	reader.Close()
}

func runAgThroughReader(pattern string, reader io.Reader, name string) {
	tempFile, err := ioutil.TempFile("", "*.tmp")
	if err != nil {
		fmt.Printf("Error creating temp file for %s: %s\n", name, err)
		return
	}
	defer os.Remove(tempFile.Name())

	if _, err = io.Copy(tempFile, reader); err != nil {
		fmt.Printf("Error copying to temp file for %s: %s\n", name, err)
		return
	}

	runAg(pattern, tempFile.Name())
	tempFile.Close()
}

func cleanupLogs(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".log") {
			if time.Since(info.ModTime()).Hours() > 168 { // 168 hours in 7 days
				fmt.Printf("Compressing and deleting old log file: %s\n", path)
				compressLog(path)
			}
		}
		return nil
	})
}

func compressLog(path string) {
	input, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening %s: %s\n", path, err)
		return
	}
	defer input.Close()

	output, err := os.Create(path + ".zstd")
	if err != nil {
		fmt.Printf("Error creating compressed file for %s: %s\n", path, err)
		return
	}
	defer output.Close()

	encoder, err := zstd.NewWriter(output)
	if err != nil {
		fmt.Printf("Error creating zstd encoder for %s: %s\n", path, err)
		return
	}
	defer encoder.Close()

	if _, err = io.Copy(encoder, input); err != nil {
		fmt.Printf("Error compressing %s: %s\n", path, err)
		return
	}

	if err = os.Remove(path); err != nil {
		fmt.Printf("Error removing original log file %s: %s\n", path, err)
		return
	}
}
