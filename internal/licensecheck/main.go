package main

import (
	"context"
	"encoding/csv"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/google/go-licenses/licenses"
	"github.com/google/licenseclassifier"
)

func main() {
	if err := execute(); err != nil {
		log.Fatal(err)
	}
}

func execute() error {
	classifier, err := licenses.NewClassifier(0.9)
	if err != nil {
		return err
	}

	f, err := os.OpenFile("LICENSE-3rdparty.csv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	libs, err := licenses.Libraries(context.Background(), classifier, ".")
	if err != nil {
		return err
	}

	// Sort libs so csv diff is neater
	sort.SliceStable(libs, func(i, j int) bool {
		return strings.Compare(libs[i].Name(), libs[j].Name()) < 0
	})

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Component", "Origin", "License", "Copyright"}); err != nil {
		return err
	}

	for _, lib := range libs {
		var (
			origin, licenseName, copyright string
		)
		if lib.LicensePath != "" {
			// Find a URL for the license file, based on the URL of a remote for the Git repository.
			repo, err := licenses.FindGitRepo(lib.LicensePath)
			if err != nil {
				// Can't find Git repo (possibly a Go Module?) - derive URL from lib name instead.
				if lURL, err := lib.FileURL(lib.LicensePath); err == nil {
					origin = lURL.String()
				}
				if b, err := ioutil.ReadFile(lib.LicensePath); err == nil {
					copyright = licenseclassifier.CopyrightHolder(string(b))
				}
			} else {
				for _, remote := range []string{"origin", "upstream"} {
					url, err := repo.FileURL(lib.LicensePath, remote)
					if err != nil {
						continue
					}
					origin = url.String()
					break
				}
			}
			if ln, _, err := classifier.Identify(lib.LicensePath); err == nil {
				licenseName = ln
			}
		}
		// Parse the project URL
		originUrlParts := strings.Split(origin, "/blob/")
		if err := w.Write([]string{lib.Name(), originUrlParts[0], licenseName, copyright}); err != nil {
			return err
		}
	}

	return nil
}
