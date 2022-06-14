package fileutil

import (
	"archive/zip"
	"errors"
	"io/ioutil"
)

// readFile reads content of the archived file
func readFile(file *zip.File) (result []byte, err error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return content, err
}

// ReadFileFromZip reads zip archive and returns content of the file by filename
func ReadFileFromZip(zipFile, filename string) (result []byte, err error) {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == filename {
			result, err := readFile(file)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	return result, errors.New("was unable to find file in zip")
}
