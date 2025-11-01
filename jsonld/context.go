package jsonld

// MangleNamespace is the default namespace URI for Mangle predicates and concepts.
const MangleNamespace = "http://mangle.datalog.org/"

// DefaultContext returns the default JSON-LD context for Mangle Atoms.
// This context provides namespace definitions for RDF vocabularies and
// Mangle-specific constant types (pairs, maps, structs).
//
// The hybrid conversion approach uses arity-based mapping:
// - Arity 0: {"@type": "predicate"}
// - Arity 1: {"@id": arg0, "@type": "predicate"}
// - Arity 2: {"@id": arg0, "predicate": arg1}
// - Arity 3+: {"@type": "predicate", "arg0": val, "arg1": val, ...}
func DefaultContext() map[string]interface{} {
	return map[string]interface{}{
		"@vocab": MangleNamespace,
		"xsd":    "http://www.w3.org/2001/XMLSchema#",

		// Generic argument properties for n-ary relations (arity 3+)
		"arg0": MangleNamespace + "arg0",
		"arg1": MangleNamespace + "arg1",
		"arg2": MangleNamespace + "arg2",
		"arg3": MangleNamespace + "arg3",
		"arg4": MangleNamespace + "arg4",
		"arg5": MangleNamespace + "arg5",
		"arg6": MangleNamespace + "arg6",
		"arg7": MangleNamespace + "arg7",
		"arg8": MangleNamespace + "arg8",
		"arg9": MangleNamespace + "arg9",

		// Special constructors for Mangle compound types
		"fn:pair": map[string]interface{}{
			"@id":        MangleNamespace + "pair",
			"@container": "@list",
		},
		"fn:map": map[string]interface{}{
			"@id":        MangleNamespace + "map",
			"@container": "@list",
		},
		"fn:struct": map[string]interface{}{
			"@id":        MangleNamespace + "struct",
			"@container": "@list",
		},
	}
}
