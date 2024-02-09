// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package perconatests

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestPrepareExporters extracts exporter from client binary's tar.gz
func TestPrepareUpdatedExporter(t *testing.T) {
	if doRun == nil || !*doRun {
		t.Skip("For manual runs only through make")
		return
	}

	if url == nil || *url == "" {
		t.Error("URL not defined")
		return
	}

	prepareExporter(*url, updatedExporterFileName)
}

func extractExporter(gzipStream io.Reader, fileName string) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("ExtractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	exporterFound := false
	for !exporterFound {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			if strings.HasSuffix(header.Name, "postgres_exporter") {
				outFile, err := os.Create(fileName)
				if err != nil {
					log.Fatalf("ExtractTarGz: Create() failed: %s", err.Error())
				}
				defer outFile.Close()
				if _, err := io.Copy(outFile, tarReader); err != nil {
					log.Fatalf("ExtractTarGz: Copy() failed: %s", err.Error())
				}

				exporterFound = true
			}
		default:
			log.Fatalf(
				"ExtractTarGz: unknown type: %d in %s",
				header.Typeflag,
				header.Name)
		}
	}
}

func prepareExporter(url, fileName string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	extractExporter(resp.Body, fileName)

	err = exec.Command("chmod", "+x", fileName).Run()
	if err != nil {
		log.Fatal(err)
	}
}
