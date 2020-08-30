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

// validateResponse validates an HTTP response against an OpenAPI spec
func (v *Validator) validateResponse(rr caddyhttp.ResponseRecorder, request *http.Request, requestValidationInput *openapi3filter.RequestValidationInput) *oapiError {

	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestValidationInput,
		Status:                 rr.Status(),
		Header:                 rr.Header(),
	}

	responseValidationInput.SetBodyBytes(rr.Buffer().Bytes())

	if v.options != nil {
		responseValidationInput.Options = &v.options.Options
	}

	v.logger.Debug(fmt.Sprintf("%#v", responseValidationInput))

	requestContext := request.Context()

	err := openapi3filter.ValidateResponse(requestContext, responseValidationInput)
	if err != nil {
		v.logger.Error(err.Error())
		// TODO: do something with different cases (switch) and return an error (overwrite http status code, if possible?)
	}

	return nil
}
