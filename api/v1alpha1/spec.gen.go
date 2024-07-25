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
