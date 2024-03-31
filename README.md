[![Go Reference](https://pkg.go.dev/badge/github.com/tigerwill90/enviro.svg)](https://pkg.go.dev/github.com/tigerwill90/enviro)
[![tests](https://github.com/tigerwill90/enviro/actions/workflows/tests.yaml/badge.svg)](https://github.com/tigerwill90/enviro/actions?query=workflow%3Atests)
[![codecov](https://codecov.io/gh/tigerwill90/enviro/branch/master/graph/badge.svg?token=09nfd7v0Bl)](https://codecov.io/gh/tigerwill90/enviro)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tigerwill90/enviro)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/tigerwill90/enviro)
# Enviro

Enviro is a Go library designed to simplify the process of loading and parsing environment variables into Go structs. It supports a wide range of field types, nested structs, custom types, and more, with an emphasis on convenience and ease of use.

## Disclaimer
Enviro is currently in a very early stage of development. As such, the API is not yet stabilized, and breaking changes may occur before we reach v1.0.0.

## Features

- **Automatic Parsing**: Automatically parse environment variables into Go structs.
- **Nested Structs**: Support for nested structs to organize your configuration logically.
- **Custom Types**: Easily handle custom types with the `ParseField` interface.
- **Flexible Tagging**: Use struct tags to specify environment variable names and options.
- **Built-in Type Support**: Out-of-the-box support for common Go types and parsing of complex types like URLs, times, and files.

## Installation

To install Enviro, use the following `go get` command:

```sh
go get -u github.com/tigerwill90/enviro
```

## Usage

Here's a quick example to show how you can use Enviro to load environment variables into a struct:

```go
package main

import (
	"github.com/tigerwill90/enviro"
	"log"
	"net/url"
	"time"
)

type Config struct {
	Port  int            `enviro:"port" envdefault:"8080"` // MYAPP_PORT=8080
	Host  string         `enviro:"host,required"`          // MYAPP_HOST=localhost
	Local *time.Location `enviro:"tz,required,omitprefix"` // TZ=America/New_York
	Debug bool           `enviro:"debug"`                  // MYAPP_DEBUG=true
	Proxy struct {
		Url     url.URL       `enviro:"url"`     // MYAPP_PROXY_URL=https://example.com
		Timeout time.Duration `enviro:"timeout"` // MYAPP_PROXY_TIMEOUT=5s
	} `enviro:"nested:proxy"`
}

func main() {
	env := enviro.New()
	env.SetEnvPrefix("MYAPP")
	cfg := Config{
		// Port: 8080, or set a default value directly in the struct
	}
	if err := env.ParseEnv(&cfg); err != nil {
		log.Fatalf("Error loading config: %s", err)
	}

	log.Printf("Loaded config: %+v", cfg)
}
```

### Struct Tags

- `enviro`: Specifies the name of the environment variable and options (e.g., `required`).
- `envopt`: Provides additional parsing options for complex types (e.g., file permissions).

## Supported Types

Enviro supports all basic Go types (`int`, `string`, `bool`, etc.), slices, maps, and any type implementing the `ParseField` interface for custom parsing logic.

## Contributing

We welcome contributions! Please feel free to submit a pull request or create an issue for bugs, feature requests, or documentation improvements.

## License

Enviro is released under the MIT License. See the bundled LICENSE file for details.