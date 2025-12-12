# API Directory

This directory contains the Protocol Buffer definitions for the services.
The structure follows the Kubernetes API group/version pattern: `api/<group>/<version>`.

## Structure

- `hello/v1`: Hello service API definitions.
- `user-center/v1`: User Center service API definitions.

## Code Generation

Code generation is managed by `buf`.
To generate the code, run:

```bash
make gen.proto
```

This will generate:
- Go code (`.pb.go`)
- gRPC code (`_grpc.pb.go`)
- Validation code (`.validate.pb.go`)
- OpenAPI specifications (`.swagger.json`)
