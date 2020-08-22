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
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
)

func (v *Validator) validateRequest(rw http.ResponseWriter, request *http.Request) *httpError {

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
		// QueryParams  url.Values
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
