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
	"io/ioutil"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// TODO: add some other functionality for wrapping kin-openapi / swagger functionality, like validation

// NoopAuthenticationFunc is an AuthenticationFunc used for mocking/disabling auth checks
func NoopAuthenticationFunc(context.Context, *openapi3filter.AuthenticationInput) error { return nil }

// readOpenAPISpecification returns the OpenAPI specification corresponding
func readOpenAPISpecification(path string) (*openapi3.Swagger, error) {

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	openapi, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(contents)
	if err != nil {
		return nil, fmt.Errorf("error loading OpenAPI specification: %s", err)
	}

	return openapi, nil
}
