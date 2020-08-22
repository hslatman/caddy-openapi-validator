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
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func init() {
	caddy.RegisterModule(ResponseValidator{})
}

// CaddyModule returns the Caddy module information.
func (ResponseValidator) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.openapi_response_validator",
		New: func() caddy.Module { return new(ResponseValidator) },
	}
}

// Provision sets up the OpenAPI Validator responder.
func (v *ResponseValidator) Provision(ctx caddy.Context) error {

	specification, err := readOpenAPISpecification(v.Filepath)
	if err != nil {
		return err
	}
	specification.Servers = nil  // TODO: enabled this; or make optional via here or options
	specification.Security = nil // TODO: enabled this; or make optional via here or options
	v.specification = specification

	// TODO: validate the specification is a valid spec?
	router := openapi3filter.NewRouter().WithSwagger(v.specification)
	v.router = router

	v.options = &validatorOptions{
		Options: openapi3filter.Options{
			ExcludeRequestBody:    false,
			ExcludeResponseBody:   false,
			IncludeResponseStatus: true,
			//AuthenticationFunc: ,
		},
		//ParamDecoder: ,
	}

	return nil
}

// ResponseValidator is used to validate OpenAPI responses to an OpenAPI specification
type ResponseValidator struct {
	specification *openapi3.Swagger
	options       *validatorOptions
	router        *openapi3filter.Router

	// TODO: options to set: enabled/disabled; server checks enabled; security checks enabled
	// TODO: add logging
	// TODO: add option to operate in inspection mode (with logging invalid requests, rather than hard blocking invalid requests)

	// The filepath to the OpenAPI (v3) specification to use
	Filepath string `json:"filepath,omitempty"`
	// The prefix to strip off when performing validation
	Prefix string `json:"prefix,omitempty"`
}

func (v *ResponseValidator) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {

	err := v.validateResponseFromContext(w, r)
	if err != nil {
		// TODO: we should generate an error response here based on some of the returned data?
		fmt.Println(err.Error())
		w.WriteHeader(err.Code)
		return nil
	}

	// If everything was OK, we continue to the next handler
	return next.ServeHTTP(w, r) // TODO: how to pass additional handlers, like other nexts?

	// TODO: can we also validate responses?
}

func (v *ResponseValidator) validateResponseFromContext(rw http.ResponseWriter, request *http.Request) *httpError {
	fmt.Println("response validation to be implemented")

	// TODO: this handler should be after an actual API call; in case of the example PetStore.go API, handled by Caddy, there's already a
	// status code. This may also be true when calling other APIs. We should make sure that we can (optionally, based on configuration), overrule
	// the return status code in case the response is not valid according to the API specification (or just log that.)

	return nil
}
