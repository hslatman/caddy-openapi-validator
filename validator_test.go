package openapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/oxtoacart/bpool"
	"go.uber.org/zap/zaptest"
)

func createValidator(t *testing.T) (*Validator, error) {

	d, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	boolValue := true
	validator := &Validator{
		Filepath:          path.Join(d, "examples/petstore.yaml"), // TODO: provide spec as string for testing purposes?
		ValidateRoutes:    &boolValue,
		ValidateRequests:  &boolValue,
		ValidateResponses: &boolValue,
	}

	// NOTE: we're performing the Provision() steps manually here, because there's a lot going on under the hood of Caddy
	validator.logger = zaptest.NewLogger(t)
	validator.bufferPool = bpool.NewBufferPool(64)
	err = validator.prepareOpenAPISpecification()
	if err != nil {
		return nil, err
	}

	return validator, nil
}

func prepareRequest(method, url string) (*http.Request, error) {
	replacer := caddy.NewReplacer()
	newContext := context.WithValue(context.Background(), caddy.ReplacerCtxKey, replacer)
	req, err := http.NewRequestWithContext(newContext, method, url, nil)
	if err != nil {
		return nil, err
	}
	//req.Header.Set("Host", "https://localhost:9443")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "*")
	req.Header.Set("User-Agent", "caddy-openapi-validator-test")

	return req, nil
}

type mockAPI struct {
}

type pet struct {
	ID int `json:"id"`
	//Name string `json:"name"`
	Tag string `json:"tag,omitempty"`
	//Additional string `json:"additional"`
}

// ServeHTTP serves a mock Pet Store API for testing purposes
func (m *mockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) error {

	// TODO: provide a bit more realistic data that actually conforms to the OpenAPI specification?
	w.Header().Set("Content-Type", "application/json")

	pet1 := pet{
		ID: 1,
		//Name: "Pet 1",
		//Additional: "this should trigger an error",
	}
	json.NewEncoder(w).Encode(pet1)

	w.WriteHeader(200)

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

func TestServeHTTP(t *testing.T) {
	v, err := createValidator(t)
	if err != nil {
		t.Fatal(err)
	}

	req, err := prepareRequest("GET", "http://localhost:9443/api/pets/1")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	mock := &mockAPI{}
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
		t.Errorf("handler returned unexpected content-type header: got %v want %v",
			recorder.Header().Get("Content-Type"), expectedContentTypeHeader)
	}

	//t.Log(fmt.Sprintf("%#v", recorder))
	//t.Fail()

	// TODO: more tests?
}
