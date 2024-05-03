# oneapi

oneapi is a tool to generate OpenAPI 3.1 specification from Go source code.

## Features

API generation:
- [x] Generate OpenAPI 3.1 schemas from Go structs
- [] Generate OpenAPI 3.1 paths from Go functions
- [] Generate Go server code from OpenAPI 3.1

Database generation:
- [] Generate SQL schema from Go structs
- [] Generate Go database layer from Go structs

## Installation

```bash
go install -v github.com/danhtran94/oneapi/cmd/oneapi

# example tests
oneapi -path "tests/models/*.go" > tests/openapi.yaml 
```