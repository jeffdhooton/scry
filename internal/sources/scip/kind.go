package scip

import scipbindings "github.com/scip-code/scip/bindings/go/scip"

// KindFromSymbol returns only the category proven by the final SCIP descriptor.
// A Type descriptor does not distinguish a class, interface, struct or alias;
// a Term does not distinguish a field, variable or Go interface method. Never
// infer those distinctions from documentation text or the symbol's spelling.
// Invalid symbols, locals, parameters and unsupported descriptors stay unknown.
func KindFromSymbol(symbol string) string {
	parsed, err := scipbindings.ParseSymbol(symbol)
	if err != nil || parsed.GetScheme() == "" || len(parsed.GetDescriptors()) == 0 {
		return ""
	}
	last := parsed.GetDescriptors()[len(parsed.GetDescriptors())-1]
	switch last.GetSuffix() {
	case scipbindings.Descriptor_Method:
		return "Method"
	case scipbindings.Descriptor_Type:
		return "Type"
	case scipbindings.Descriptor_Term:
		return "Term"
	case scipbindings.Descriptor_Namespace:
		return "Namespace"
	default:
		return ""
	}
}

func symbolKind(si *scipbindings.SymbolInformation) string {
	if si.GetKind() != scipbindings.SymbolInformation_UnspecifiedKind {
		return si.GetKind().String()
	}
	if kind := KindFromSymbol(si.GetSymbol()); kind != "" {
		return kind
	}
	return si.GetKind().String()
}

// Priority prevents an early reference or unspecified declaration from hiding
// authoritative information emitted later in the streaming index.
type symbolPriority uint8

const (
	synthesizedSymbol symbolPriority = iota + 1
	descriptorSymbol
	authoritativeSymbol
)
