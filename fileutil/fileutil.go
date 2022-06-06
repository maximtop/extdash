package fileutil

import (
	"archive/zip"
	"errors"
	"io/ioutil"
)

func readFile(file *zip.File) (result []byte, err error) {
	reader, err := file.Open()
	if err != nil {
		return result, err
	}
	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return result, err
	}
	return content, err
}

func ReadFileFromZip(zipFile, filename string) (result []byte, err error) {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return result, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == filename {
			result, err := readFile(file)
			if err != nil {
				return result, err
			}
			return result, err
		}
	}

	return result, errors.New("was unable to find file in zip")
}
