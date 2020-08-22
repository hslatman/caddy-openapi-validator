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

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func (v *Validator) validateResponse(rr caddyhttp.ResponseRecorder, request *http.Request) *httpError {

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

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    request,
		PathParams: pathParams,
		Route:      route,
		// QueryParams  url.Values
	}

	if v.options != nil {
		requestValidationInput.Options = &v.options.Options
		requestValidationInput.ParamDecoder = v.options.ParamDecoder
	}

	v.logger.Debug(fmt.Sprintf("%#v", requestValidationInput))

	// TODO: use ResponseRecorder functionality? And/or do this in the defer?

	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestValidationInput,
		Status:                 rr.Status(),
		Header:                 rr.Header(),
	}

	responseValidationInput.SetBodyBytes(rr.Buffer().Bytes())

	v.logger.Debug(fmt.Sprintf("bytes: %#v", rr.Buffer().Bytes()))

	if v.options != nil {
		responseValidationInput.Options = &v.options.Options
	}

	v.logger.Debug(fmt.Sprintf("%#v", responseValidationInput))

	requestContext := request.Context()

	err = openapi3filter.ValidateResponse(requestContext, responseValidationInput)
	if err != nil {
		v.logger.Error(err.Error())
		// TODO: do something with different cases (switch) and return an error (overwrite http status code, if possible?)
	}

	return nil
}
