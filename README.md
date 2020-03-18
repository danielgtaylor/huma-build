# Huma Build

A Docker build helper for [Huma](https://github.com/danielgtaylor/huma).

## Features

- Builds your Huma-based service using Alpine
- Optionally generates a CLI using [openapi-cli-generator](https://github.com/danielgtaylor/openapi-cli-generator)
- Optionally generates SDKs using [openapi-generator](https://openapi-generator.tech/)
  - See [available generators](https://openapi-generator.tech/docs/generators#client-generators)

## Usage

Usage requires two files: a `.huma.yaml` configuration and a `Dockerfile`. Examples:

```yaml
service: your-service
command: go test && go install
cli:
  name: your-service-cli
  command: go build
sdk-languages:
  - go
  - python
  - javascript
```

Note that CLI support requires a `cli` folder with a `main.go` that generates all needed code via `go generate`.

```Dockerfile
FROM danielgtaylor/huma-build as build
COPY . .
RUN huma-build

FROM alpine as deploy
WORKDIR /service
COPY --from=build /go/bin/your-service /usr/bin/
COPY --from=build /huma/out/*.zip ./downloads/
ENTRYPOINT ["your-service"]
```

You can then build and run the service!

```sh
# Build it!
$ docker build -t your-service .

# Run it!
$ docker run your-service
```

Check it out at http://localhost:8888/docs.
