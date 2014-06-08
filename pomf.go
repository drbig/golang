// Simplest pomf.se uploader
// dRbiG 2014
//

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type PomfResponse struct {
	Success bool
	Error   string
	Files   []map[string]interface{}
}

const (
	FIELD    = "files[]"
	UPLOAD   = "http://pomf.se/upload.php"
	DOWNLOAD = "http://a.pomf.se/"
	SIZE     = 52428800
)

var (
	buffer *bytes.Buffer
	ack    PomfResponse
	client *http.Client
)

func upload(src io.Reader, filename string) bool {
	buffer.Reset()
	writer := multipart.NewWriter(buffer)

	part, err := writer.CreateFormFile(FIELD, filename)
	if err != nil {
		panic(err)
	}

	size, err := io.Copy(part, src)
	if err != nil {
		panic(err)
	}
	if size > SIZE {
		fmt.Println("ERROR: Input too large")
		return false
	}

	err = writer.Close()
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", UPLOAD, buffer)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	buffer.Reset()
	_, err = buffer.ReadFrom(res.Body)
	if err != nil {
		panic(err)
	}
	res.Body.Close()

	var ack PomfResponse
	err = json.Unmarshal(buffer.Bytes(), &ack)
	if err != nil {
		panic(err)
	}

	if ack.Success {
		fmt.Printf("%s%s\n", DOWNLOAD, ack.Files[0]["url"])
		return true
	} else {
		fmt.Printf("ERROR: %s\n", ack.Error)
		return false
	}
}

func main() {
	buffer = &bytes.Buffer{}
	client = &http.Client{}

	if len(os.Args) == 1 {
		if upload(os.Stdin, "stdin") {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	} else {
		failed := 0

		for _, arg := range os.Args[1:] {
			path, err := filepath.Abs(arg)
			if err != nil {
				panic(err)
			}

			filename := filepath.Base(path)
			fmt.Printf("%s: ", filename)

			handle, err := os.Open(path)
			if err != nil {
				panic(err)
			}
			info, err := handle.Stat()
			if err != nil {
				panic(err)
			}
			if info.Size() > SIZE {
				fmt.Println("ERROR: File too large")
				failed++
				continue
			}
			if !upload(handle, filename) {
				failed++
			}
			handle.Close()
		}

		if failed > 0 {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}
}
