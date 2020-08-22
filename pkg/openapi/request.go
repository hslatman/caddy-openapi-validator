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
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"go.uber.org/zap"
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

func NoopAuthenticationFunc(context.Context, *openapi3filter.AuthenticationInput) error { return nil }

// Provision sets up the OpenAPI Validator responder.
func (v *RequestValidator) Provision(ctx caddy.Context) error {

	v.logger = ctx.Logger(v)

	specification, err := readOpenAPISpecification(v.Filepath)
	if err != nil {
		return err
	}
	specification.Servers = nil  // TODO: enabled this; or make optional via here or options
	specification.Security = nil // TODO: enabled this; or make optional via here or options
	v.specification = specification

	// TODO: validate the specification is a valid spec? Is actually performed via WithSwagger, but can break the program, so we might need to to this in Validate()
	router := openapi3filter.NewRouter().WithSwagger(v.specification)
	v.router = router

	v.options = &validatorOptions{
		Options: openapi3filter.Options{
			ExcludeRequestBody:    false,
			ExcludeResponseBody:   false,
			IncludeResponseStatus: true,
			AuthenticationFunc:    NoopAuthenticationFunc, // TODO: can we provide an actual one? And how?
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
	logger        *zap.Logger

	// TODO: options to set: enabled/disabled; server checks enabled; security checks enabled
	// TODO: add option to operate in inspection mode (with logging invalid requests, rather than hard blocking invalid requests; i.e. don't respond)

	// The filepath to the OpenAPI (v3) specification to use
	Filepath string `json:"filepath,omitempty"`
	// The prefix to strip off when performing validation
	Prefix string `json:"prefix,omitempty"`
}

// ServeHTTP is the Caddy handler for serving HTTP requests
func (v *RequestValidator) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {

	err := v.validateRequestFromContext(w, r)
	if err != nil {
		// TODO: we should generate an error response here based on some of the returned data? in what format? (configured or via accept headers?)
		v.logger.Error(err.Error())
		w.WriteHeader(err.Code)
		return nil
	}

	// If everything was OK, we continue to the next handler
	return next.ServeHTTP(w, r)
}

func (v *RequestValidator) validateRequestFromContext(rw http.ResponseWriter, request *http.Request) *httpError {

	url, err := determineRequestURL(request, v.Prefix)
	if err != nil {
		return &httpError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}
	method := request.Method
	route, pathParams, err := v.router.FindRoute(method, url)

	// No route found for the request
	if err != nil {
		switch e := err.(type) {
		case *openapi3filter.RouteError:
			// The requested path doesn't match the server, path or anything else.
			// TODO: switch between cases based on the e.Reason string? Some are not found, some are invalid method, etc.
			return &httpError{
				Code:    http.StatusBadRequest,
				Message: e.Reason,
			}
		default:
			// Fallback for unexpected or unimplemented cases
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

	v.logger.Debug(fmt.Sprintf("%#v", validationInput)) // TODO: output something a little bit nicer?

	// TODO: can we (in)validate additional query parameters? The default behavior does not seem to take additional params into account

	requestContext := request.Context() // TODO: add things to the request context, if required?

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
			// Fallback for unexpected or unimplemented cases
			return &httpError{
				Code:     http.StatusInternalServerError,
				Message:  fmt.Sprintf("error validating request: %s", err),
				Internal: err,
			}
		}
	}

	return nil
}
