package util

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func CreateFileIfNotExist(filePath string) error {
	if _, err := os.Stat(filePath); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		_, err := os.Create(filePath)
		if err != nil {
			return errors.New(fmt.Sprintf("error in creating: %s", filePath))
		}
	} else {
		return errors.Wrap(err, fmt.Sprintf("error in checking if file exists: %s", filePath))
	}
	return nil
}

// unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func SaveListToAfile(outputPath string, fileName string, dataList []string) error {
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {

		file, err := os.OpenFile(filepath.Join(outputPath, fileName), os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			errors.Wrap(err, fmt.Sprintf("error in creating file: %s", fileName))
		}

		datawriter := bufio.NewWriter(file)

		for _, data := range dataList {
			_, _ = datawriter.WriteString(data + "\n")
		}

		datawriter.Flush()
		file.Close()
	} else {
		return errors.Wrap(err, fmt.Sprintf("error directory doesn't exist: %s", outputPath))
	}
	return nil
}

func Zip(src string, dest string, zipDirName string) (string, error) {

	dpath := filepath.Join(dest, zipDirName+".zip")
	archive, err := os.Create(dpath)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error in creating: %s", dpath))
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)

	err = filepath.Walk(src, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}
		relPath := strings.TrimPrefix(filePath, src)
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		fsFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(zipFile, fsFile)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error in adding files to the zip dir: %s", src))
	}

	err = zipWriter.Close()
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error in closing zip dir: %s", dpath))
	}

	return dpath, nil
}
