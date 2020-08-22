package openapi

import "fmt"

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
