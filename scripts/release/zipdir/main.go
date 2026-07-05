package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: zipdir <archive.zip> <base-dir> <entry-dir>")
		os.Exit(2)
	}

	archivePath := os.Args[1]
	baseDir := os.Args[2]
	entryDir := os.Args[3]

	if err := zipDirectory(archivePath, baseDir, entryDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func zipDirectory(archivePath, baseDir, entryDir string) error {
	root := filepath.Join(baseDir, entryDir)

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive %s: %w", archivePath, err)
	}
	defer archive.Close()

	writer := zip.NewWriter(archive)
	defer writer.Close()

	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("create zip header for %s: %w", path, err)
		}
		relativePath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return fmt.Errorf("make relative path for %s: %w", path, err)
		}
		header.Name = filepath.ToSlash(relativePath)
		header.Method = zip.Deflate

		destination, err := writer.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("add %s to archive: %w", path, err)
		}

		source, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		defer source.Close()

		if _, err := io.Copy(destination, source); err != nil {
			return fmt.Errorf("copy %s to archive: %w", path, err)
		}
		return nil
	})
}
