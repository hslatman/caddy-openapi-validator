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
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func init() {
	caddy.RegisterModule(Validator{})
}

// CaddyModule returns the Caddy module information.
func (Validator) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.openapi_validator",
		New: func() caddy.Module { return new(Validator) },
	}
}

// Provision sets up the OpenAPI Validator responder.
func (v *Validator) Provision(ctx caddy.Context) error {

	specification := v.getOpenAPISpecification(v.Filepath)
	v.specification = specification

	router := openapi3filter.NewRouter().WithSwagger(v.specification)
	v.router = router

	options := nil // TODO: handle other options to pass down
	v.options = options

	return nil
}

// Validator is used to validate OpenAPI requests to an OpenAPI specification
type Validator struct {
	specification *openapi3.Swagger
	options       *ValidatorOptions
	router        *openapi3filter.Router

	// TODO: options to set: enabled/disabled; server checks enabled; security checks enabled; filepath to OpenAPI
	Filepath string `json:"filepath,omitempty"`
}

// ValidatorOptions  are optinos to customize request validation.
// These are passed through to openapi3filter.
type ValidatorOptions struct {
	Options      openapi3filter.Options
	ParamDecoder openapi3filter.ContentParameterDecoder
	UserData     interface{}
}

type httpError struct {
	Code     int         `json:"-"`
	Message  interface{} `json:"message"`
	Internal error       `json:"-"` // Stores the error returned by an external dependency
}

func (he *httpError) Error() string {
	if he.Internal != nil {
		return fmt.Sprintf("code=%d, message=%v, internal=%v", he.Code, he.Message, he.Internal)
	}

	return fmt.Sprintf("code=%d, message=%v", he.Code, he.Message)
}

func (v *Validator) ServeHTTP(rw http.ResponseWriter, r *http.Request, next caddyhttp.Handler) {

	err := v.validateRequestFromContext(rw, r, router, options)
	if err != nil {
		// TODO: we should generate an error response here based on some of the returned data?
		fmt.Println(err.Error())
		w.WriteHeader(err.Code)
		return nil
	}

	// If everything was OK, we continue to the next handler
	next(rw, r)

	// TODO: can we also validate responses?
}

// getOpenAPISpecification returns the OpenAPI specification corresponding
func (v *Validator) getOpenAPISpecification(path string) (*openapi3.Swagger, error) {

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

func (v *Validator) validateRequestFromContext(rw http.ResponseWriter, request *http.Request) *httpError {

	route, pathParams, err := v.router.FindRoute(request.Method, request.URL)

	// We failed to find a matching route for the request.
	if err != nil {
		switch e := err.(type) {
		case *openapi3filter.RouteError:
			// We've got a bad request, the path requested doesn't match either server, or path, or something.
			return &httpError{
				Code:    http.StatusBadRequest,
				Message: e.Reason,
			}
		default:
			// This should never happen today, but if our upstream code changes, we don't want to crash the server, so handle the unexpected error.
			return &httpError{
				Code:    http.StatusInternalServerError,
				Message: fmt.Sprintf("error validating route: %s", err.Error()),
			}
		}
	}

	validationInput := &openapi3filter.RequestValidationInput{
		Request:    request,
		PathParams: pathParams,
		Route:      route,
	}

	// // Pass the Chi context into the request validator, so that any callbacks which it invokes make it available.
	// ctx := request.Context()
	// requestContext := context.WithValue(context.Background(), chiContextKey, ctx)

	// if v.options != nil {
	// 	validationInput.Options = &options.Options
	// 	validationInput.ParamDecoder = options.ParamDecoder
	// 	requestContext = context.WithValue(requestContext, userDataKey, options.UserData)
	// }

	requestContext := request.Context()

	err = openapi3filter.ValidateRequest(requestContext, validationInput)
	if err != nil {
		switch e := err.(type) {
		case *openapi3filter.RequestError:
			// We've got a bad request
			// Split up the verbose error by lines and return the first one
			// openapi errors seem to be multi-line with a decent message on the first
			errorLines := strings.Split(e.Error(), "\n")
			return &httpError{
				Code:     http.StatusBadRequest,
				Message:  errorLines[0],
				Internal: err,
			}
		case *openapi3filter.SecurityRequirementsError:
			return &httpError{
				Code:     http.StatusForbidden,
				Message:  e.Error(),
				Internal: err,
			}
		default:
			// This should never happen today, but if our upstream code changes,
			// we don't want to crash the server, so handle the unexpected error.
			return &httpError{
				Code:     http.StatusInternalServerError,
				Message:  fmt.Sprintf("error validating request: %s", err),
				Internal: err,
			}
		}
	}

	return nil
}
