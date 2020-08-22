package openapi

import (
	"fmt"
	"io/ioutil"

	"github.com/getkin/kin-openapi/openapi3"
)

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
