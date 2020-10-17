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

// validateRequest validates an HTTP requests according to an OpenAPI spec
func (v *Validator) validateRequest(rw http.ResponseWriter, r *http.Request, validationInput *openapi3filter.RequestValidationInput) *oapiError {

	v.logger.Debug(fmt.Sprintf("%#v", validationInput)) // TODO: output something a little bit nicer?

	// TODO: can we (in)validate additional query parameters? The default behavior does not seem to take additional params into account

	requestContext := r.Context() // TODO: add things to the request context, if required?

	err := openapi3filter.ValidateRequest(requestContext, validationInput)
	if err != nil {
		switch e := err.(type) {
		case *openapi3filter.RequestError:
			// A bad request with a verbose error; splitting it and taking the first line
			errorLines := strings.Split(e.Error(), "\n")
			return &oapiError{
				Code:     http.StatusBadRequest,
				Message:  errorLines[0],
				Internal: err,
			}
		case *openapi3filter.SecurityRequirementsError:
			if v.shouldValidateSecurity() {
				return &oapiError{
					Code:     http.StatusForbidden, // TOOD: is this the right code? The validator is not the authorizing party.
					Message:  formatFullError(e),
					Internal: err,
				}
			}
		default:
			// Fallback for unexpected or unimplemented cases
			return &oapiError{
				Code:     http.StatusInternalServerError,
				Message:  fmt.Sprintf("error validating request: %s", err),
				Internal: err,
			}
		}
	}

	return nil
}
