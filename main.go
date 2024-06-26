package main

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

// ANSI color codes
const (
	Magenta = "\033[35m"
	Pink    = "\033[38;5;206m" // 256-color mode pink
	Reset   = "\033[0m"        // Reset all attributes
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gl [ag-options] PATTERN [dir]")
		os.Exit(1)
	}

	// Collect command-line arguments
	args := os.Args[1:]
	agOptions := []string{}
	var pattern, dir string

	// Determine if the last argument is a directory
	if len(args) > 1 && !strings.HasPrefix(args[len(args)-1], "-") {
		if info, err := os.Stat(args[len(args)-1]); err == nil && info.IsDir() {
			dir = args[len(args)-1]
			args = args[:len(args)-1] // Remove directory from args
		}
	}

	// Pattern is the last argument now
	if len(args) > 0 {
		pattern = args[len(args)-1]
		agOptions = args[:len(args)-1]
	}

	if pattern == "" {
		fmt.Println("Pattern must be provided.")
		os.Exit(1)
	}

	// If no directory specified, use current working directory
	if dir == "" {
		dir, _ = os.Getwd()
	}

	// fmt.Printf("Pattern: %s\nDirectory: %s\nAG Options: %v\n", pattern, dir, agOptions)

	// Walk through the directory structure
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			switch {
			case strings.HasSuffix(path, ".zip"), strings.HasSuffix(path, ".gz"), strings.HasSuffix(path, ".tgz"), strings.HasSuffix(path, ".zstd"):
				handleCompressedFile(path, pattern, agOptions)
			default:
				runAg(pattern, path, agOptions)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %q: %v\n", dir, err)
	}

	cleanupLogs(dir)
}

func runAg(pattern, path string, options []string) {
	cmdArgs := append(options, pattern, path)
	cmd := exec.Command("ag", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return
			}
		}
		fmt.Printf("Error running ag on %s: %s\n", path, string(output))
		return
	}
	fmt.Printf("%sResults for %s%s:\n%s\n", Magenta, path, Reset, string(output))
}

func handleCompressedFile(path, pattern string, options []string) {
	// fmt.Printf("Handling compressed file: %s\n", path)
	ext := filepath.Ext(path)

	var reader io.ReadCloser
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening file %s: %s\n", path, err)
		return
	}
	defer file.Close()

	switch ext {
	case ".gz":
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
			runAgThroughReader(pattern, reader, f.Name, options)
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

	runAgThroughReader(pattern, reader, path, options)
	reader.Close()
}

// runAgThroughReader reads data from a reader and directly pipes it to the `ag` command
func runAgThroughReader(pattern string, reader io.Reader, name string, options []string) {
	// Prepare the ag command with given options
	cmdArgs := append(options, pattern)
	cmd := exec.Command("ag", cmdArgs...)
	cmd.Stdin = reader // Set the command's Stdin to the reader

	// Execute the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		// if exit status is 1, it means no results were found
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return
			}
		}
		fmt.Printf("Error running ag through reader for %s: %s\n", name, err)
		return
	}

	// Print results
	fmt.Printf("%sResults for %s%s (from stream):\n%s\n", Magenta, name, Reset, string(output))
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
