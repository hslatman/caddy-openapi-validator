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
	"net/url"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func init() {
	caddy.RegisterModule(RequestValidator{})
}

// CaddyModule returns the Caddy module information.
func (RequestValidator) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.openapi_request_validator",
		New: func() caddy.Module { return new(RequestValidator) },
	}
}

// Provision sets up the OpenAPI Validator responder.
func (v *RequestValidator) Provision(ctx caddy.Context) error {

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

// RequestValidator is used to validate OpenAPI requests to an OpenAPI specification
type RequestValidator struct {
	specification *openapi3.Swagger
	options       *validatorOptions
	router        *openapi3filter.Router

	// TODO: options to set: enabled/disabled; server checks enabled; security checks enabled

	// The filepath to the OpenAPI (v3) specification to use
	Filepath string `json:"filepath,omitempty"`
	// The prefix to strip off when performing validation
	Prefix string `json:"prefix,omitempty"`
}

func (v *RequestValidator) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {

	err := v.validateRequestFromContext(w, r)
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

func (v *RequestValidator) validateRequestFromContext(rw http.ResponseWriter, request *http.Request) *httpError {

	// TODO: determine whether this is (still) required when we're checking the servers (again)
	url, err := url.ParseRequestURI(request.URL.String()[len(v.Prefix):])
	if err != nil {
		return &httpError{
			Code:    http.StatusBadRequest,
			Message: "error while cutting off prefix",
		}
	}
	method := request.Method
	route, pathParams, err := v.router.FindRoute(method, url)

	// No route found for the request
	if err != nil {
		switch e := err.(type) {
		case *openapi3filter.RouteError:
			// The requested path doesn't match the server, or path, or anything else.
			return &httpError{
				Code:    http.StatusBadRequest,
				Message: e.Reason,
			}
		default:
			// Provide a fallback in case something unexpected happens
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

	if v.options != nil {
		validationInput.Options = &v.options.Options
		validationInput.ParamDecoder = v.options.ParamDecoder
	}

	// TODO: can we invalidate additional query parameters? The default behavior does not seem to take additional params into account

	// TODO: adapt my code below, used within a project that uses Chi, to use a context, if we need that ... ?
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
			// A bad request with a verbose error; splitting it and taking the first
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
			// Provide a fallback in case something unexpected happens
			return &httpError{
				Code:     http.StatusInternalServerError,
				Message:  fmt.Sprintf("error validating request: %s", err),
				Internal: err,
			}
		}
	}

	return nil
}
