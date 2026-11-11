package catalog

import _ "embed"

// Data is generated from the built-in template catalog before packaging.
//
//go:embed catalog.json
var Data []byte
