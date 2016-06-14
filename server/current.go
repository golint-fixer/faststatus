// Copyright 2016 Jesse Allen. All rights reserved
// Released under the MIT license found in the LICENSE file.

// Server provides a RESTful API for faststatus resources.
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/lazyengineering/faststatus/resource"
)

// Current encapsulates the api endpoint for managing current resource status
type Current struct {
	db *bolt.DB
}

func NewCurrent(dbPath string) (*Current, error) {
	s := new(Current)
	err := s.init(dbPath)
	if err != nil {
		return nil, fmt.Errorf("Error creating new Current: %v", err)
	}
	return s, nil
}

func (s *Current) init(dbPath string) error {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return fmt.Errorf("Error initializing database, %q: %v", dbPath, err)
	}
	s.db = db
	return nil
}

func (s *Current) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// first, separate by path, then method

	ids, err := idsFromPath(r.URL.Path)
	if err != nil {
		error404(w, r)
	}

	switch r.Method {
	case http.MethodGet:
		s.getResource(w, r, ids)
	case http.MethodPut:
		s.putResource(w, r)
	case http.MethodPost:
		s.postResource(w, r)
	case http.MethodDelete:
		s.deleteResource(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// return
func error404(w http.ResponseWriter, r *http.Request) {
	switch textOrJson(r.Header[http.CanonicalHeaderKey("Accept")]) {
	case "text/plain":
		http.Error(w, "Resource Not Found", http.StatusNotFound)
	case "application/json":
		http.Error(w, "[]", http.StatusNotFound)
	}
}

func error500(w http.ResponseWriter, r *http.Request) {
	switch textOrJson(r.Header[http.CanonicalHeaderKey("Accept")]) {
	case "text/plain":
		http.Error(w, "Server Error", http.StatusInternalServerError)
	case "application/json":
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func textOrJson(accepts []string) string {
	for _, a := range accepts {
		switch a {
		case "application/json":
			return "application/json"
		case "text/plain":
			fallthrough
		case "*/*":
			return "text/plain"
		}
	}
	return "text/plain"
}

func idsFromPath(path string) ([]uint64, error) {
	var ids []uint64
	raw := strings.Split(path, "/")
	for _, s := range raw {
		if s == "" {
			continue
		}
		id, err := strconv.ParseUint(s, 16, 64)
		if err != nil {
			return ids, fmt.Errorf("Error parsing ids, %q: %v", path, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func encode_json(w io.Writer, rs []resource.Resource) error {
	return json.NewEncoder(w).Encode(rs)
}

func encode_text(w io.Writer, rs []resource.Resource) error {
	for _, r := range rs {
		_, err := fmt.Fprintln(w, r.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func encoder(accept string) func(io.Writer, []resource.Resource) error {
	switch accept {
	case "application/json":
		return encode_json
	case "text/plain":
		fallthrough
	default:
		return encode_text
	}
}

// expects an empty request, returns the resource
func (s *Current) getResource(w http.ResponseWriter, r *http.Request, ids []uint64) {
	if len(ids) == 0 {
		error404(w, r)
		return
	}
	rch := make(chan []byte)
	done := make(chan struct{})
	defer close(done)
	go s.db.View(func(tx *bolt.Tx) error {
		defer close(rch)
		b := tx.Bucket([]byte("resources"))
		for _, id := range ids {
			raw := b.Get([]byte(strconv.FormatUint(id, 16)))
			select {
			case rch <- raw:
			case <-done:
				return nil
			}
		}
		return nil
	})

	resources := []resource.Resource{}
	for raw := range rch {
		if raw == nil {
			continue
		}
		rc := new(resource.Resource)
		err := rc.UnmarshalJSON(raw)
		if err != nil {
			error500(w, r)
			return
		}
		resources = append(resources, *rc)
	}
	if len(resources) == 0 {
		error404(w, r)
		return
	}

	tmp := new(bytes.Buffer)
	err := encoder(textOrJson(r.Header[http.CanonicalHeaderKey("Accept")]))(tmp, resources)
	if err != nil {
		error500(w, r)
		return
	}
	tmp.WriteTo(w)
}

// expects a valid resource, returns the new/updated resource. ID in body must match the ID in the URL
func (s *Current) putResource(w http.ResponseWriter, r *http.Request) {
}

//
func (s *Current) deleteResource(w http.ResponseWriter, r *http.Request) {
}

func (s *Current) postResource(w http.ResponseWriter, r *http.Request) {
}
