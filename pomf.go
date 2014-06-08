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
)

const (
	FIELD    = "files[]"
	UPLOAD   = "http://pomf.se/upload.php"
	DOWNLOAD = "http://a.pomf.se/"
)

type PomfResponse struct {
	Success bool
	Error   string
	Files   []map[string]interface{}
}

func main() {
	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)

	part, err := writer.CreateFormFile(FIELD, "stdin")
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(part, os.Stdin)
	if err != nil {
		panic(err)
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

	client := &http.Client{}
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

	if !ack.Success {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", ack.Error)
		os.Exit(1)
	} else {
		fmt.Printf("%s%s\n", DOWNLOAD, ack.Files[0]["url"])
		os.Exit(0)
	}
}
