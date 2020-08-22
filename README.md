# Caddy OpenAPI Validator

A [Caddy](https://caddyserver.com/) module that validates requests against an OpenAPI specification

## Description

The Caddy OpenAPI Validator module is a Caddy HTTP handler that validates requests against an OpenAPI specification.

## Usage

Include the HTTP handler as a Caddy module:

```golang
import (
	_ "github.com/hslatman/caddy-openapi-validator/pkg/openapi"
)
```

Configure the OpenAPI Validator handler as one of the handlers to be executed first by Caddy:

```json
    ...
        "handle": [
            {
            "handler": "openapi_validator",
            "filepath": "examples/petstore.yaml"
            }
        ]    
    ...
```

## TODO

* Add logging
* Improve the example with an actual implementation and more handlers
* Add other options
* Look into response validation
* Look into ways to specify the error nicely
* Add support for remote API specs?
