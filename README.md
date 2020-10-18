# Caddy OpenAPI Validator (WIP)

A [Caddy](https://caddyserver.com/) module that validates requests and responses against an [OpenAPI](https://www.openapis.org/) specification.

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

The simplest way to use the OpenAPI Validator HTTP handler is by using `xcaddy`:

```bash
$ xcaddy build v2.1.1 --with github.com/hslatman/caddy-openapi-validator
```

Alternatively, the HTTP handler can be included as a Caddy module as follows:

```golang
import (
	_ "github.com/hslatman/caddy-openapi-validator"
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
                "validate_responses": true,
                "validate_servers": true,
                "validate_security": true,
                "path_prefix_to_be_trimmed": "",
                "additional_servers": [""],
                "enforce": true,
                "log": true
            }
        ]
    ...
```

The OpenAPI Validator handler should be called before an actual API is called.
The configuration shown above shows the default settings.
The `filepath` configuration is required; without it, or when pointing to a non-existing file, the module won't be loaded.

## Example

An example of the OpenAPI Validatory HTTP handler in use can be found [here](https://github.com/hslatman/caddy-openapi-validator-example).

## Notes

This project is currently a small POC with the intention to grow it along with my other projects using Go, OpenAPI and Caddy.
I only recently started using Caddy, so there may be some rough edges to iron out when more use cases present themselves.

## TODO

A small and incomplete list of potential things to implement, improve and think about:

* Add more tests for the OpenAPI Validator functionality and configuration.
* Improve Caddyfile handling (e.g. add more subdirectives).
* Add an example that uses an HTTP proxy/fcgi configuration.
* Look into ways to specify the error nicely, instead of just logging it (e.g. return error message(s) in specific format) and/or integrate properly with how Caddy handlers errors.
* Look into if (and how) the Validator can be used outside of Caddy as an alternative (i.e. a more generic middleware).
* Add option to specify servers in addition to the one in the OpenAPI specification for server checks.
