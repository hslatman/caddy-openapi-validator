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
