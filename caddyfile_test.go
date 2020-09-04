package openapi

import (
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func TestSimpleCaddyfile(t *testing.T) {
	input := `openapi_validator path/to/openapi.yaml`

	d := caddyfile.NewTestDispenser(input)

	v := &Validator{}

	err := v.UnmarshalCaddyfile(d)
	if err != nil {
		t.Error(err)
	}

	if v.Filepath != "path/to/openapi.yaml" {
		t.Errorf("got: %s, want: path/to/openapi.yaml", v.Filepath)
	}
}
