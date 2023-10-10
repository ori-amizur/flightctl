// Package v1alpha1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.15.0 DO NOT EDIT.
package v1alpha1

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"AA==",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
