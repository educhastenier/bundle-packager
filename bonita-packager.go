package main

import (
	"archive/zip"
	"errors"
	"fmt"
	cp "github.com/otiai10/copy"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	_sep = os.PathSeparator
)

func main() {
	// Try to find a Bonita zip file inside :
	matches, err := filepath.Glob("src" + string(_sep) + "Bonita*.zip")
	if err != nil {
		panic(err)
	}
	if len(matches) == 0 || !Exists(filepath.Join("src", "my-application")) {
		fmt.Println("Please place in src/ folder:")
		fmt.Println(" * ZIP file of Bonita Tomcat Bundle (Eg. BonitaCommunity-2023.1.zip)")
		fmt.Println(" * my-application/ folder containing all artifacts of your application")
		fmt.Println("and then re-run this program")
		return
	}
	if Exists("output") {
		fmt.Println("Cleaning 'output' directory")
		if err := os.RemoveAll("output"); err != nil {
			panic(err)
		}
	}
	bundleNameAndPath := matches[0]
	bundleName := bundleNameAndPath[4:strings.Index(bundleNameAndPath, ".zip")] // until end of string
	fmt.Println("Unpacking Bonita Tomcat bundle" + bundleName + ".zip")
	unzipFile(bundleNameAndPath, "output")
	fmt.Println("Unpacking Bonita WAR file")
	unzipFile(filepath.Join("output", bundleName, "server", "webapps", "bonita.war"), filepath.Join("output", bundleName, "server", "webapps", "bonita"))
	fmt.Println("Removing unpacked Bonita WAR file")
	if err := os.Remove(filepath.Join("output", bundleName, "server", "webapps", "bonita.war")); err != nil {
		panic(err)
	}
	fmt.Println("Copying your custom application inside Bonita")
	err = cp.Copy(filepath.Join("src", "my-application"), filepath.Join("output", bundleName, "server", "webapps", "bonita", "WEB-INF", "classes", "my-application"))
	if err != nil {
		panic(err)
	}
	fmt.Println("Re-packing Bonita bundle containing your application")
	err = zipDirectory(filepath.Join("output", bundleName+"-custom-application.zip"), filepath.Join("output", bundleName), bundleName)
	if err != nil {
		panic(err)
	}
	fmt.Println("Done. Self-contained application available: " + bundleName + "-application.zip")
}

func unzipFile(zipFile string, outputDir string) {
	archive, err := zip.OpenReader(zipFile)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(outputDir, f.Name)
		//fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(outputDir)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			//fmt.Println("creating directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
}

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func zipDirectory(zipFilename string, baseDir string, baseInZip string) error {
	outFile, err := os.Create(zipFilename)
	if err != nil {
		return err
	}

	w := zip.NewWriter(outFile)

	if err := addFilesToZip(w, baseDir, baseInZip); err != nil {
		_ = outFile.Close()
		return err
	}

	if err := w.Close(); err != nil {
		_ = outFile.Close()
		return errors.New("Warning: closing zipfile writer failed: " + err.Error())
	}

	if err := outFile.Close(); err != nil {
		return errors.New("Warning: closing zipfile failed: " + err.Error())
	}

	return nil
}

func addFilesToZip(w *zip.Writer, basePath, baseInZip string) error {
	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		fullfilepath := filepath.Join(basePath, file.Name())
		if _, err := os.Stat(fullfilepath); os.IsNotExist(err) {
			// ensure the file exists. For example a symlink pointing to a non-existing location might be listed but not actually exist
			continue
		}

		if file.Mode()&os.ModeSymlink != 0 {
			// ignore symlinks alltogether
			continue
		}

		if file.IsDir() {
			if err := addFilesToZip(w, fullfilepath, filepath.Join(baseInZip, file.Name())); err != nil {
				return err
			}
		} else if file.Mode().IsRegular() {
			dat, err := ioutil.ReadFile(fullfilepath)
			if err != nil {
				return err
			}
			f, err := w.Create(filepath.Join(baseInZip, file.Name()))
			if err != nil {
				return err
			}
			_, err = f.Write(dat)
			if err != nil {
				return err
			}
		} else {
			// we ignore non-regular files because they are scary
		}
	}
	return nil
}
