package petstore

import (
	"encoding/json"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(PetStore{})
}

// PetStore struct keeping module data
type PetStore struct {
}

type pet struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
	//Additional string `json:"additional"`
}

// CaddyModule defines the PetStore module
func (PetStore) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.petstore_api_example",
		New: func() caddy.Module { return new(PetStore) },
	}
}

// ServeHTTP serves a simple (and currently incomplete) Pet Store API
func (p *PetStore) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {

	// TODO: provide a bit more realistic data that actually conforms to the OpenAPI specification?
	w.Header().Set("Content-Type", "application/json")

	pet1 := pet{
		ID:   1,
		Name: "Pet 1",
		//Additional: "this should trigger an error",
	}
	json.NewEncoder(w).Encode(pet1)

	w.WriteHeader(200)

	return next.ServeHTTP(w, r)
}
