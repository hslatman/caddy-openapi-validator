// Copyright 2020 Herman Slatman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/oxtoacart/bpool"
	"go.uber.org/zap/zaptest"
)

func createValidator(t *testing.T) (*Validator, error) {

	boolValue := true
	validator := &Validator{
		Filepath:              "examples/petstore.yaml",
		ValidateRoutes:        &boolValue,
		ValidateRequests:      &boolValue,
		ValidateResponses:     &boolValue,
		ValidateServers:       &boolValue,
		ValidateSecurity:      &boolValue,
		PathPrefixToBeTrimmed: "",
		AdditionalServers:     []string{"https://localhost:9443/api", "http://localhost:9443/api"},
		Enforce:               &boolValue,
		Log:                   &boolValue,
	}

	// NOTE: we're performing the Provision() steps manually here, because there's a lot going on under the hood of Caddy
	validator.logger = zaptest.NewLogger(t)
	validator.bufferPool = bpool.NewBufferPool(64)
	err := validator.prepareOpenAPISpecification()
	if err != nil {
		return nil, err
	}

	return validator, nil
}

func replaceValidator(v *Validator) (*Validator, error) {
	new := &Validator{
		Filepath:              v.Filepath,
		ValidateRoutes:        v.ValidateRoutes,
		ValidateRequests:      v.ValidateRequests,
		ValidateResponses:     v.ValidateResponses,
		ValidateServers:       v.ValidateServers,
		ValidateSecurity:      v.ValidateSecurity,
		PathPrefixToBeTrimmed: v.PathPrefixToBeTrimmed,
		AdditionalServers:     v.AdditionalServers,
		Enforce:               v.Enforce,
		Log:                   v.Log,
		logger:                v.logger,
		bufferPool:            v.bufferPool,
	}

	err := new.prepareOpenAPISpecification()
	if err != nil {
		return nil, err
	}

	return new, nil
}

func prepareRequest(method, url string) (*http.Request, error) {
	replacer := caddy.NewReplacer()
	newContext := context.WithValue(context.Background(), caddy.ReplacerCtxKey, replacer)
	req, err := http.NewRequestWithContext(newContext, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "*")
	req.Header.Set("User-Agent", "caddy-openapi-validator-test")

	return req, nil
}

type mockWrongAPI struct {
}

type mockAPI struct {
}

type notFoundAPI struct {
}

type wrongPet struct {
	ID  int    `json:"id"`
	Tag string `json:"tag,omitempty"`
}

type pet struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
}

// ServeHTTP serves an API with wrong responses
func (m *mockWrongAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) error {

	w.Header().Set("Content-Type", "application/json")

	pet1 := wrongPet{
		ID: 1,
	}

	json.NewEncoder(w).Encode(pet1)

	w.WriteHeader(200)

	return nil
}

// ServeHTTP serves a mock Pet Store API for testing purposes
func (m *mockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) error {

	w.Header().Set("Content-Type", "application/json")

	pet1 := pet{
		ID:   1,
		Name: "Pet 1",
	}
	json.NewEncoder(w).Encode(pet1)

	w.WriteHeader(200)

	return nil
}

// ServeHTTP serves a 404 API
func (m *notFoundAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) error {

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(404)

	return nil
}

// TODO: test provisioning stage?

func TestValidate(t *testing.T) {
	v, err := createValidator(t)
	if err != nil {
		t.Error(err)
	}

	// TODO: make configuration for testing a bit nicer
	bValue := false
	v.ValidateRoutes = &bValue
	err = v.Validate()
	if err == nil {
		t.Error("validator should fail when no route validation is performed, but requests or responses are validated")
	}

	bValue = true
	v.ValidateRoutes = &bValue
	err = v.Validate()
	if err != nil {
		t.Error(err)
	}
}

func TestServerValidation(t *testing.T) {
	v, err := createValidator(t)
	if err != nil {
		t.Fatal(err)
	}

	mock := &mockAPI{}

	req, err := prepareRequest("GET", "http://some-unknown-host:9443/api/pets/1")
	if err != nil {
		t.Error(err)
	}

	recorder := httptest.NewRecorder()

	err = v.ServeHTTP(recorder, req, mock)
	if err == nil {
		t.Error("expected an error while enforcing server validation")
	}

	// Disable server validation and set /api as URL path prefix to be trimmed
	// so that we can keep the same URL in the request
	bValue := false
	v.ValidateServers = &bValue
	v.PathPrefixToBeTrimmed = "/api"

	n, err := replaceValidator(v)
	if err != nil {
		t.Fatal(err)
	}

	if n.shouldValidateServers() {
		t.Error("server validation should be off")
	}

	req, err = prepareRequest("GET", "http://some-unknown-host:9443/api/pets/1")
	if err != nil {
		t.Fatal(err)
	}

	recorder = httptest.NewRecorder()

	err = n.ServeHTTP(recorder, req, mock)
	if err != nil {
		t.Error(err)
	}
}

func TestSecurityValidation(t *testing.T) {
	v, err := createValidator(t)
	if err != nil {
		t.Fatal(err)
	}

	// Override the default petstore.yaml with the secured one
	v.Filepath = "examples/petstore-secured.yaml"
	v, err = replaceValidator(v)
	if err != nil {
		t.Fatal(err)
	}

	// Use a secured version of the PetStore API; HTTP Basic Authentication is enabled
	if !(v.Filepath == "examples/petstore-secured.yaml") {
		t.Error("wrong filepath set for security validation test")
	}

	if !v.shouldValidateSecurity() {
		t.Error("security validation should be on")
	}

	mock := &mockAPI{}

	// This request does not have an HTTP Basic Authentication (Authorization) header and should fail
	req, err := prepareRequest("GET", "http://localhost:9443/api/pets/1")
	if err != nil {
		t.Error(err)
	}

	recorder := httptest.NewRecorder()

	err = v.ServeHTTP(recorder, req, mock)
	if err == nil {
		t.Error("expected an error while enforcing security validation")
	}

	// Prepare the same request, but this time we're adding HTTP Basic Authentication
	req, err = prepareRequest("GET", "http://localhost:9443/api/pets/1")
	if err != nil {
		t.Error(err)
	}

	req.Header.Set("Authorization", "Basic Y2FkZHk6b3BlbmFwaQ==")

	recorder = httptest.NewRecorder()

	err = v.ServeHTTP(recorder, req, mock)
	if err != nil {
		t.Error(err)
	}

	// We turn security validation off; now requests can be executed without security set
	bValue := false
	v.ValidateSecurity = &bValue

	n, err := replaceValidator(v)
	if err != nil {
		t.Fatal(err)
	}

	if n.shouldValidateSecurity() {
		t.Error("security validation should be off")
	}

	req, err = prepareRequest("GET", "http://localhost:9443/api/pets/1")
	if err != nil {
		t.Fatal(err)
	}

	recorder = httptest.NewRecorder()

	err = n.ServeHTTP(recorder, req, mock)
	if err != nil {
		t.Error(err)
	}
}

func TestAdditionalServers(t *testing.T) {
	v, err := createValidator(t)
	if err != nil {
		t.Fatal(err)
	}

	mock := &mockAPI{}

	req, err := prepareRequest("GET", "http://localhost:9443/api/pets/1")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	err = v.ServeHTTP(recorder, req, mock)
	if err != nil {
		t.Error(err)
	}

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	v.AdditionalServers = []string{}

	n, err := replaceValidator(v)
	if err != nil {
		t.Fatal(err)
	}

	req, err = prepareRequest("GET", "http://localhost:9443/api/pets/1")
	if err != nil {
		t.Fatal(err)
	}

	recorder = httptest.NewRecorder()

	err = n.ServeHTTP(recorder, req, mock)
	if err == nil {
		t.Error("expected an error without additional servers specified")
	}
}

func TestNonExistingRoute(t *testing.T) {
	v, err := createValidator(t)
	if err != nil {
		t.Fatal(err)
	}

	// The request has a typo in the route; it does not exist
	req, err := prepareRequest("GET", "http://localhost:9443/api/petz/1")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	mock := &mockWrongAPI{}

	err = v.ServeHTTP(recorder, req, mock)
	if err == nil { // TODO: add tests with non-enforcement (i.e. no error expected)
		t.Error("expected an error while enforcing route validation")
	}

	if status := recorder.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	t.Log(fmt.Sprintf("%#v", recorder))
}

// TODO: can we validate query parameters?
// func TestWrongQueryParameter(t *testing.T) {

// 	v, err := createValidator(t)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// The request has a typo in the route; it does not exist
// 	req, err := prepareRequest("GET", "http://localhost:9443/api/pets?limis=10#bla")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	recorder := httptest.NewRecorder()

// 	mock := &mockAPI{}

// 	err = v.ServeHTTP(recorder, req, mock)
// 	if err == nil { // TODO: add tests with non-enforcement (i.e. no error expected)
// 		t.Error("expected an error while enforcing route validation")
// 	}

// 	if status := recorder.Code; status != http.StatusNotFound {
// 		t.Errorf("handler returned wrong status code: got %v want %v",
// 			status, http.StatusNotFound)
// 	}

// 	t.Log(fmt.Sprintf("%#v", recorder))

// 	//t.Fail()
// }

func TestWrongSchemaInResponse(t *testing.T) {

	v, err := createValidator(t)
	if err != nil {
		t.Fatal(err)
	}

	req, err := prepareRequest("GET", "http://localhost:9443/api/pets/1")
	if err != nil {
		t.Fatal(err)
	}

	mock := &mockWrongAPI{}

	recorder := httptest.NewRecorder()

	err = v.ServeHTTP(recorder, req, mock)
	if err == nil {
		t.Error("expected an error while enforcing response validation")
	}

	if status := recorder.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	t.Log(fmt.Sprintf("%#v", recorder))
}

func TestEmptyResponse(t *testing.T) {

	v, err := createValidator(t)
	if err != nil {
		t.Fatal(err)
	}

	mock := &notFoundAPI{}

	req, err := prepareRequest("GET", "http://localhost:9443/api/pets/2")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	err = v.ServeHTTP(recorder, req, mock)
	if err != nil {
		t.Error(err)
	}

	if status := recorder.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	t.Log(fmt.Sprintf("%#v", recorder))
}

func TestServeHTTP(t *testing.T) {
	v, err := createValidator(t)
	if err != nil {
		t.Fatal(err)
	}

	mock := &mockAPI{}

	req, err := prepareRequest("GET", "http://localhost:9443/api/pets/1")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	err = v.ServeHTTP(recorder, req, mock)
	if err != nil {
		t.Error(err)
	}

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedContentTypeHeader := "application/json"
	if contentTypeHeader := recorder.Header().Get("Content-Type"); contentTypeHeader != expectedContentTypeHeader {
		t.Errorf("handler returned unexpected content-type header: got '%v' want '%v'",
			recorder.Header().Get("Content-Type"), expectedContentTypeHeader)
	}

	t.Log(fmt.Sprintf("%#v", recorder))
}
