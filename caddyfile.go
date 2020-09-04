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
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

const (
	// TKOpenAPIValidator is token for the main directive
	TKOpenAPIValidator = "openapi_validator"
	// TKFilepath is token for the subdirective that points to the OpenAPI filepath
	TKFilepath = "filepath"
	// TKValidateRoutes is token for the subdirective that sets route validation on or off
	TKValidateRoutes = "routes"
	// TKValidateRequests is token for the subdirective that sets requests validation on or off
	TKValidateRequests = "requests"
	// TKValidateResponses is token for the subdirective that sets response validation on or off
	TKValidateResponses = "responses"
)

// parseCaddyfile is used as the entrypoint for unmarshalling the Caddyfile for openapi_validator
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var validator *Validator
	err := validator.UnmarshalCaddyfile(h.Dispenser)
	return validator, err
}

// UnmarshalCaddyfile parses (part of) the Caddyfile and configures a Validator
func (v *Validator) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {

	v.Filepath = ""

	// Set all validations to true
	boolValue := true
	v.ValidateRoutes = &boolValue
	v.ValidateRequests = &boolValue
	v.ValidateResponses = &boolValue

	d.Next()
	args := d.RemainingArgs()
	if len(args) == 1 {
		d.NextArg()
		v.Filepath = d.Val()
	}

	for nest := d.Nesting(); d.NextBlock(nest); {
		token := d.Val()
		switch token {
		case TKFilepath:
			if !d.NextArg() {
				return d.Err("missing path to OpenAPI specification")
			}
			v.Filepath = d.Val()
			if d.NextArg() {
				return d.ArgErr()
			}
		// TODO: add handling of the other directives: routes, requests, responses and new ones
		default:
			return d.Errf("unrecognized token: '%s'", token)
		}
	}

	// Check the bare minimum configuration; return error if it's not OK
	if v.Filepath == "" {
		return d.Err("missing path to OpenAPI specification")
	}

	return nil
}
