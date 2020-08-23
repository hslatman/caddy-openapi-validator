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

	"github.com/oxtoacart/bpool"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(Validator{})
}

// Validator is used to validate OpenAPI requests and responses to an OpenAPI specification
type Validator struct {
	specification *openapi3.Swagger
	options       *validatorOptions
	router        *openapi3filter.Router
	logger        *zap.Logger
	bufferPool    *bpool.BufferPool

	// TODO: options to set: enabled/disabled; server checks enabled; security checks enabled
	// TODO: add option to operate in inspection mode (with logging invalid requests, rather than hard blocking invalid requests; i.e. don't respond)

	// The filepath to the OpenAPI (v3) specification to use
	Filepath string `json:"filepath,omitempty"`
	// The prefix to strip off when performing validation
	Prefix string `json:"prefix,omitempty"`
	// Indicates whether routes should be validated
	// When ValidateRequests or ValidateResponses is true, ValidateRoutes should also be true
	// Default is true
	ValidateRoutes *bool `json:"validate_routes,omitempty"`
	// Indicates whether request validation should be enabled
	// Default is true
	ValidateRequests *bool `json:"validate_requests,omitempty"`
	// Indicates whether request validation should be enabled
	// Default is true
	ValidateResponses *bool `json:"validate_responses,omitempty"`
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

	v.bufferPool = bpool.NewBufferPool(64)

	return nil
}

// Validate validates the configuration of the Validator
func (v *Validator) Validate() error {

	shouldValidateRoutes := v.ValidateRoutes == nil || *v.ValidateRoutes
	shouldValidateRequests := v.ValidateRequests == nil || *v.ValidateRequests
	shouldValidateResponses := v.ValidateResponses == nil || *v.ValidateResponses

	if (shouldValidateRequests || shouldValidateResponses) && !shouldValidateRoutes {
		return fmt.Errorf("route validation can't be disabled when validation of requests or responses is enabled")
	}

	return nil
}

// ServeHTTP is the Caddy handler for serving HTTP requests
func (v *Validator) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {

	var requestValidationInput *openapi3filter.RequestValidationInput = nil
	var oerr *oapiError = nil

	if v.ValidateRoutes == nil || *v.ValidateRoutes {
		requestValidationInput, oerr = v.validateRoute(r)
		if oerr != nil {
			// TODO: we should generate an error response here based on some of the returned data? in what format? (configured or via accept headers?)
			v.logger.Error(oerr.Error())
			w.WriteHeader(oerr.Code)
			return nil // TODO: return the actual error here?
		}
	}

	if v.ValidateRequests == nil || *v.ValidateRequests {
		oerr := v.validateRequest(w, r, requestValidationInput)
		if oerr != nil {
			// TODO: we should generate an error response here based on some of the returned data? in what format? (configured or via accept headers?)
			v.logger.Error(oerr.Error())
			w.WriteHeader(oerr.Code)
			return nil // TODO: return the actual error here?
		}
	}

	// In case we shouldn't validate responses, we're going to execute the next handler and return early
	if v.ValidateResponses != nil && !*v.ValidateResponses {
		err := next.ServeHTTP(w, r)
		if err != nil {
			return err
		}
		return nil
	}

	// In case we should validate responses, we need to record the response and read that before returning the response
	buffer := v.bufferPool.Get()
	defer v.bufferPool.Put(buffer)

	shouldBuffer := func(status int, header http.Header) bool {
		// TODO: add logic for performing buffering vs. not doing it; what is logical to do?
		return true
	}
	recorder := caddyhttp.NewResponseRecorder(w, buffer, shouldBuffer)

	// Continue down the handler stack, recording the response, so that we can work with it afterwards
	err := next.ServeHTTP(recorder, r)
	if err != nil {
		return err
	}

	if !recorder.Buffered() {
		// TODO: do we need to do something with this?
		//return nil
	}

	// TODO: can we validate additional/superfluous fields? And make that configurable? The validator configured now does not seem to do that.
	oerr = v.validateResponse(recorder, r, requestValidationInput)
	if oerr != nil {
		// TODO: we should generate an error response here based on some of the returned data? in what format? (configured or via accept headers?)
		// TODO: we might also want to send this information in some other way, like setting a header, only logging, or in response format itself
		v.logger.Error(oerr.Error())
		w.WriteHeader(oerr.Code)
		return nil // TODO: return the actual error here?
	}

	// TODO: we've wrapped the handler chain and are at the end; if there are errors, we may want to override the response and its
	// status code. This may also be true when calling other APIs (not just for the PetStore API example).
	// We should make sure that we can (optionally, based on configuration), overrule the return status code in case the response
	// is not valid according to the API specification (or just log that.)

	recorder.WriteResponse() // Actually writes the response (after having buffered the bytes); the easy way

	return nil
}