package fileutil

import (
	"archive/zip"
	"fmt"
	"io"

	"github.com/AdguardTeam/golibs/errors"
)

const (
	_        = iota
	KB int64 = 1 << (10 * iota)
	MB
)

// readFile reads content of the archived file.
func readFile(file *zip.File) (result []byte, err error) {
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("[readFile] error occurred on opening file: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, reader.Close()) }()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("[readFile] error occurred on reading file: %w", err)
	}

	return content, err
}

// ReadFileFromZip reads zip archive and returns content of the file by filename.
func ReadFileFromZip(zipFile, filename string) (result []byte, err error) {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return nil, fmt.Errorf("[ReadFileFromZip] error occurred on opening zip file: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, reader.Close()) }()

	for _, file := range reader.File {
		if file.Name == filename {
			result, err := readFile(file)
			if err != nil {
				return nil, fmt.Errorf("[ReadFileFromZip] error occurred on reading file: %w", err)
			}

			return result, nil
		}
	}

	return result, fmt.Errorf("was unable to find file: %s in zip", filename)
}
