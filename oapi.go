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
	"fmt"
	"net/url"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// TODO: add some other functionality for wrapping kin-openapi / swagger functionality, like validation

// NoopAuthenticationFunc is an AuthenticationFunc used for mocking/disabling auth checks
func NoopAuthenticationFunc(context.Context, *openapi3filter.AuthenticationInput) error { return nil }

// readOpenAPISpecification returns the OpenAPI specification corresponding
func readOpenAPISpecification(path string) (*openapi3.Swagger, error) {

	var openapi *openapi3.Swagger
	uri, err := url.Parse(path)
	if err == nil {
		openapi, err = openapi3.NewSwaggerLoader().LoadSwaggerFromURI(uri)
		if err != nil {
			return nil, fmt.Errorf("error loading OpenAPI specification: %s", err)
		}
	} else {
		p := path
		_, err := os.Stat(p)
		if err != nil || !os.IsExist(err) {
			return nil, err
		}
		openapi, err = openapi3.NewSwaggerLoader().LoadSwaggerFromFile(p)
		if err != nil {
			return nil, fmt.Errorf("error loading OpenAPI specification: %s", err)
		}
	}

	if openapi == nil { // fallback to an error in case openapi is nil
		return nil, fmt.Errorf("loading OpenAPI specification failed")
	}

	return openapi, nil
}