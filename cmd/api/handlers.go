package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type CodeResponse struct {
	Code string `json:"code"`
}

func (app *application) getIndexHandler(w http.ResponseWriter, r *http.Request) {
	app.templates.ExecuteTemplate(w, "index", nil)
}

func (app *application) postImageHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// Write straight to disk
	err := r.ParseMultipartForm(0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// We always want to remove the multipart file as we're copying
	// the contents to another file anyway
	defer func() {
		if remErr := r.MultipartForm.RemoveAll(); remErr != nil {
			// Log error?
		}
	}()

	// Start reading multi-part file under id "fileupload"
	f, _, err := r.FormFile("image")
	if err != nil {
		if err == http.ErrMissingFile {
			http.Error(w, "Request did not contain a file", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		return
	}
	defer f.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	code, err := app.aiService.ExtractCodeFromImage(buf.Bytes())

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	json.NewEncoder(w).Encode(code)
}
