package petstore

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(PetStore{})
}

type PetStore struct {
}

func (PetStore) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.petstore_api_example",
		New: func() caddy.Module { return new(PetStore) },
	}
}

func (p *PetStore) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {

	w.Write([]byte("There may be pets here; or not."))
	w.WriteHeader(200)

	return nil
}
