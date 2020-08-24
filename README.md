# Caddy OpenAPI Validator (WIP)

A (WIP) [Caddy](https://caddyserver.com/) module that validates requests and responses against a (local) [OpenAPI](https://www.openapis.org/) specification.

## Description

The OpenAPI Validator module is a [Caddy](https://caddyserver.com/) HTTP handler that validates requests and responses against an OpenAPI specification.
The [kin-openapi](https://github.com/getkin/kin-openapi) `openapi3filter` library performs the actual validation.
The handler in this repository is a small wrapper for the functionality provided by `openapi3filter`, with basic configuration and integrations options for Caddy. 
The project is currently in POC stage, meaning that much of it can, and likely will, change.

The request/response flow is as follows:

* When a request arrives, the Validator will look for a valid route in the provided OpenAPI specification and validate the request against the schema.
* The request is then passed on to the next HTTP handler in the chain and the Validator will wait for its response.
* After capturing the response, the Validator will validate the response to be valid.
* If no errors occurred during the validation, the response will be returned.

## Usage

Include the HTTP handler as a Caddy module:

```golang
import (
	_ "github.com/hslatman/caddy-openapi-validator/pkg/openapi"
)
```

Configure the OpenAPI Validator handler as one of the handlers to be executed by Caddy (in [JSON config](https://caddyserver.com/docs/json/) format):

```json
    ...
        "handle": [
            {
                "handler": "openapi_validator",
                "filepath": "examples/petstore.yaml",
                "validate_routes": true,
                "validate_requests": true,
                "validate_responses": true
            }
        ]
    ...
```

The OpenAPI Validator handler should be called before an actual API is called.
The configuration shown above shows the default settings.
The `filepath` configuration is required; without it, or when pointing to a non-existing file, the module won't be loaded.

## Example

This repository contains an example of using the OpenAPI Validator with the `Swagger Petstore` specification.
A minimal (and incomplete) implementation of the API is provided in `internal/petstore/petstore.go`, which only exists for demo purposes.
The `config.json` file is a Caddy configuration file in JSON format.
It configures Caddy to serve the PetStore API with OpenAPI validation, TLS and logging enabled on https://localhost:9443/api.
The example can be run like below:

```bash
# run the main command directly
$ go run cmd/main.go run --config config.json

# compile and run the server
$ go build cmd/main.go
$ ./main run --config config.json
```

The (currently incomplete) API can then be accessed via https://localhost:9443/api/pets/1. 

## Notes

This project is currently a small POC with the intention to grow it along with my other projects using Go, OpenAPI and Caddy.
I only recently started using Caddy, so there may be some rough edges to iron out when more use cases present themselves.

## TODO

A small and incomplete list of potential things to implement, improve and think about:

* Add tests for the OpenAPI Validator functionality and configuration
* Improve the example with more (and correct) handlers
* Add an example that uses an HTTP proxy/fcgi configuration
* Add other options, including security validation and server override
* Look into ways to specify the error nicely, instead of just logging it (e.g. return error message(s) in specific format)
* Add support for remote API specs?
