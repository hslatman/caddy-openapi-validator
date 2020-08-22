package openapi

import "github.com/getkin/kin-openapi/openapi3filter"

// validatorOptions  are optinos to customize request validation.
// These are passed through to openapi3filter.
type validatorOptions struct {
	Options      openapi3filter.Options
	ParamDecoder openapi3filter.ContentParameterDecoder
	UserData     interface{}
}