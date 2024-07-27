// Package v1alpha1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.3.0 DO NOT EDIT.
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
