// Laravel facade -> backing class resolver.
//
// scip-php captures `Auth::user()` as a ref to
// `Illuminate/Support/Facades/Auth#user().` — i.e. the facade class itself,
// not the manager class that actually implements the call. An agent asking
// "where is `AuthManager::user` called?" gets the wrong answer because every
// real call site is hidden behind the facade.
//
// This resolver closes the gap. After scip-php has populated the store, we
// walk every SymbolRecord, identify Illuminate facade method symbols
// (`Illuminate/Support/Facades/<X>#<method>().`), look up the matching
// backing-class method via a hardcoded facade -> manager map, and emit
// synthetic ref edges from each facade method occurrence to the backing
// method symbol. The store is updated in place.
//
// What we don't try to handle:
//
//   - Custom user-defined facades. The map covers framework facades only.
//     A user facade resolver would have to walk the project's facade classes
//     and parse `getFacadeAccessor()`. Worth doing if a real codebase needs it.
//   - Service container bindings (`app(Foo::class)`). The calibration found
//     hoopless_crm has zero of these in its providers, so deferred.
//   - Eloquent relationship traversal (`$activity->user->name`). High effort,
//     low marginal value.
//
// The hardcoded map is sourced from `Illuminate/Support/Facades/<X>::
// getFacadeAccessor()` return values in laravel/framework. Each entry maps
// the facade FQN to one or more candidate backing classes — many facades
// front a manager AND a contract (e.g. Auth fronts both AuthManager and
// the Guard contract). We try each candidate in order and emit edges to
// every match found in the store.
package php

import (
	"fmt"
	"strings"

	"github.com/jeffdhooton/scry/internal/store"
)

// facadeBackings maps Illuminate facade descriptors to candidate backing
// class descriptors. Format: "Illuminate/Support/Facades/<X>" -> []string of
// "Illuminate/<Manager>" descriptors. The leading scheme/manager/pkg/version
// is stripped — we match against descriptor suffixes.
//
// Sources: laravel/framework v11 facade source. Some facades intentionally
// have multiple targets (manager + contract); the resolver tries them in
// order and emits edges to every match found.
var facadeBackings = map[string][]string{
	"Illuminate/Support/Facades/Auth": {
		"Illuminate/Auth/AuthManager",
		"Illuminate/Contracts/Auth/Factory",
		"Illuminate/Contracts/Auth/Guard",
		"Illuminate/Contracts/Auth/StatefulGuard",
	},
	"Illuminate/Support/Facades/Cache": {
		"Illuminate/Cache/CacheManager",
		"Illuminate/Cache/Repository",
		"Illuminate/Contracts/Cache/Factory",
		"Illuminate/Contracts/Cache/Repository",
	},
	"Illuminate/Support/Facades/Config": {
		"Illuminate/Config/Repository",
		"Illuminate/Contracts/Config/Repository",
	},
	"Illuminate/Support/Facades/Cookie": {
		"Illuminate/Cookie/CookieJar",
		"Illuminate/Contracts/Cookie/Factory",
		"Illuminate/Contracts/Cookie/QueueingFactory",
	},
	"Illuminate/Support/Facades/Crypt": {
		"Illuminate/Encryption/Encrypter",
		"Illuminate/Contracts/Encryption/Encrypter",
	},
	"Illuminate/Support/Facades/DB": {
		"Illuminate/Database/DatabaseManager",
		"Illuminate/Database/Connection",
	},
	"Illuminate/Support/Facades/Date": {
		"Illuminate/Support/DateFactory",
	},
	"Illuminate/Support/Facades/Event": {
		"Illuminate/Events/Dispatcher",
		"Illuminate/Contracts/Events/Dispatcher",
	},
	"Illuminate/Support/Facades/File": {
		"Illuminate/Filesystem/Filesystem",
	},
	"Illuminate/Support/Facades/Gate": {
		"Illuminate/Auth/Access/Gate",
		"Illuminate/Contracts/Auth/Access/Gate",
	},
	"Illuminate/Support/Facades/Hash": {
		"Illuminate/Hashing/HashManager",
		"Illuminate/Contracts/Hashing/Hasher",
	},
	"Illuminate/Support/Facades/Http": {
		"Illuminate/Http/Client/Factory",
	},
	"Illuminate/Support/Facades/Lang": {
		"Illuminate/Translation/Translator",
		"Illuminate/Contracts/Translation/Translator",
	},
	"Illuminate/Support/Facades/Log": {
		"Illuminate/Log/LogManager",
		"Illuminate/Log/Logger",
		"Psr/Log/LoggerInterface",
	},
	"Illuminate/Support/Facades/Mail": {
		"Illuminate/Mail/MailManager",
		"Illuminate/Mail/Mailer",
		"Illuminate/Contracts/Mail/Factory",
		"Illuminate/Contracts/Mail/Mailer",
	},
	"Illuminate/Support/Facades/Notification": {
		"Illuminate/Notifications/ChannelManager",
		"Illuminate/Contracts/Notifications/Factory",
		"Illuminate/Contracts/Notifications/Dispatcher",
	},
	"Illuminate/Support/Facades/Password": {
		"Illuminate/Auth/Passwords/PasswordBrokerManager",
		"Illuminate/Contracts/Auth/PasswordBrokerFactory",
		"Illuminate/Contracts/Auth/PasswordBroker",
	},
	"Illuminate/Support/Facades/Queue": {
		"Illuminate/Queue/QueueManager",
		"Illuminate/Contracts/Queue/Factory",
		"Illuminate/Contracts/Queue/Queue",
	},
	"Illuminate/Support/Facades/Redirect": {
		"Illuminate/Routing/Redirector",
	},
	"Illuminate/Support/Facades/Redis": {
		"Illuminate/Redis/RedisManager",
		"Illuminate/Contracts/Redis/Factory",
	},
	"Illuminate/Support/Facades/Request": {
		"Illuminate/Http/Request",
	},
	"Illuminate/Support/Facades/Response": {
		"Illuminate/Routing/ResponseFactory",
		"Illuminate/Contracts/Routing/ResponseFactory",
	},
	"Illuminate/Support/Facades/Route": {
		"Illuminate/Routing/Router",
	},
	"Illuminate/Support/Facades/Schema": {
		"Illuminate/Database/Schema/Builder",
	},
	"Illuminate/Support/Facades/Session": {
		"Illuminate/Session/SessionManager",
		"Illuminate/Session/Store",
		"Illuminate/Contracts/Session/Session",
	},
	"Illuminate/Support/Facades/Storage": {
		"Illuminate/Filesystem/FilesystemManager",
		"Illuminate/Filesystem/FilesystemAdapter",
		"Illuminate/Contracts/Filesystem/Factory",
		"Illuminate/Contracts/Filesystem/Filesystem",
	},
	"Illuminate/Support/Facades/URL": {
		"Illuminate/Routing/UrlGenerator",
		"Illuminate/Contracts/Routing/UrlGenerator",
	},
	"Illuminate/Support/Facades/Validator": {
		"Illuminate/Validation/Factory",
		"Illuminate/Contracts/Validation/Factory",
	},
	"Illuminate/Support/Facades/View": {
		"Illuminate/View/Factory",
		"Illuminate/Contracts/View/Factory",
	},
	"Illuminate/Support/Facades/Bus": {
		"Illuminate/Bus/Dispatcher",
		"Illuminate/Contracts/Bus/Dispatcher",
		"Illuminate/Contracts/Bus/QueueingDispatcher",
	},
	"Illuminate/Support/Facades/Broadcast": {
		"Illuminate/Broadcasting/BroadcastManager",
		"Illuminate/Contracts/Broadcasting/Factory",
		"Illuminate/Contracts/Broadcasting/Broadcaster",
	},
	"Illuminate/Support/Facades/Artisan": {
		"Illuminate/Contracts/Console/Kernel",
	},
}

// FacadeStats summarizes what the resolver did, for logging.
type FacadeStats struct {
	FacadesScanned int // distinct facade methods seen
	EdgesEmitted   int // synthetic ref occurrences written
}

// RunFacadeResolver scans the store for Illuminate facade method symbols
// and emits ref edges to each matching backing-class method. Safe to call
// after scip.Parse and the non-PSR-4 walker have flushed.
func RunFacadeResolver(st *store.Store) (FacadeStats, error) {
	stats := FacadeStats{}

	// Build descriptor -> []symbolID lookup for every backing class method
	// the map cares about. We do this in two passes over the symbol space:
	//
	//   pass 1: gather facade method symbols (descriptors starting with
	//           "Illuminate/Support/Facades/<X>#")
	//   pass 2: gather backing method symbols (descriptors starting with
	//           any of the configured backing-class prefixes)
	//
	// Then for each facade method we look up its name in the backing-method
	// index and emit ref edges. The two passes share an iteration with a
	// branch on the descriptor prefix.

	// facadeMethods: facadeFQN ("Illuminate/Support/Facades/Auth") -> method name -> facade symbol id
	facadeMethods := map[string]map[string]string{}
	// backingMethods: backingFQN -> method name -> backing symbol id
	backingMethods := map[string]map[string]string{}

	// Index every backing FQN in the configured map for fast prefix matching.
	backingFQNs := map[string]bool{}
	for _, list := range facadeBackings {
		for _, b := range list {
			backingFQNs[b] = true
		}
	}

	err := st.IterateAllSymbols(func(rec *store.SymbolRecord) error {
		descriptor := descriptorOf(rec.Symbol)
		if descriptor == "" {
			return nil
		}
		// Facade method? Descriptor of the form "Illuminate/Support/Facades/X#method()."
		if strings.HasPrefix(descriptor, "Illuminate/Support/Facades/") {
			class, method := splitMethodDescriptor(descriptor)
			if class == "" || method == "" {
				return nil
			}
			if _, ok := facadeBackings[class]; !ok {
				return nil
			}
			if facadeMethods[class] == nil {
				facadeMethods[class] = map[string]string{}
			}
			facadeMethods[class][method] = rec.Symbol
			return nil
		}
		// Backing method?
		for fqn := range backingFQNs {
			if strings.HasPrefix(descriptor, fqn+"#") {
				_, method := splitMethodDescriptor(descriptor)
				if method == "" {
					return nil
				}
				if backingMethods[fqn] == nil {
					backingMethods[fqn] = map[string]string{}
				}
				backingMethods[fqn][method] = rec.Symbol
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return stats, fmt.Errorf("iterate symbols: %w", err)
	}

	w := st.NewWriter()
	defer w.Flush()

	// For each facade method we found, look up its name in every candidate
	// backing class. If the backing method exists in the store, iterate the
	// facade method's refs and emit a synthetic ref to the backing method at
	// the same position. If the backing method does NOT exist in the store
	// but the candidate is in our backing list, synthesize the symbol record
	// and the refs anyway — `scry refs AuthManager::user` should still find
	// the call sites even if AuthManager isn't directly indexed.
	for facadeFQN, methods := range facadeMethods {
		candidates := facadeBackings[facadeFQN]
		for method, facadeSymbolID := range methods {
			stats.FacadesScanned++
			for _, backingFQN := range candidates {
				backingSymbolID := backingMethods[backingFQN][method]
				if backingSymbolID == "" {
					// Synthesize one. Pick the package + version from the
					// facade's id so the synthesized id sits in the same
					// package keyspace and joins the existing data.
					backingSymbolID = synthesizeBackingSymbolID(facadeSymbolID, backingFQN, method)
					_ = w.PutSymbol(&store.SymbolRecord{
						Symbol:      backingSymbolID,
						DisplayName: method,
						Kind:        "External",
					})
					if backingMethods[backingFQN] == nil {
						backingMethods[backingFQN] = map[string]string{}
					}
					backingMethods[backingFQN][method] = backingSymbolID
				}

				// Walk every ref to the facade method and mirror it onto the
				// backing method. Defs we leave alone — the def is on the
				// facade class itself, not the backing class.
				_ = st.IterateRefs(facadeSymbolID, func(occ *store.OccurrenceRecord) error {
					mirror := *occ
					mirror.Symbol = backingSymbolID
					mirror.IsDefinition = false
					if err := w.PutOccurrence(&mirror); err != nil {
						return err
					}
					stats.EdgesEmitted++
					return nil
				})
			}
		}
	}
	return stats, nil
}

// descriptorOf strips the SCIP scheme/manager/pkg/version prefix and returns
// the trailing descriptor token. SCIP ids look like
// "scip-php composer <pkg> <version> <descriptor>"; the descriptor is the
// last space-separated token (descriptors never contain spaces in any
// scheme we care about).
func descriptorOf(symbolID string) string {
	if i := strings.LastIndex(symbolID, " "); i >= 0 {
		return symbolID[i+1:]
	}
	return symbolID
}

// splitMethodDescriptor parses "Foo/Bar#method()." into ("Foo/Bar", "method").
// Returns ("", "") if the descriptor is not a method.
func splitMethodDescriptor(descriptor string) (class, method string) {
	hash := strings.Index(descriptor, "#")
	if hash < 0 {
		return "", ""
	}
	class = descriptor[:hash]
	rest := descriptor[hash+1:]
	// Method descriptors end with "()." optionally followed by parameter
	// info like ".($param)". We want everything before the first "(".
	if i := strings.Index(rest, "("); i >= 0 {
		method = rest[:i]
	} else {
		// Field/property: "$name." — not a method.
		return "", ""
	}
	if method == "" {
		return "", ""
	}
	return class, method
}

// synthesizeBackingSymbolID builds a SCIP id for a backing method that
// scip-php never emitted. We crib the package/version from the facade's id
// so the synthetic id sits in the same package keyspace.
func synthesizeBackingSymbolID(facadeSymbolID, backingFQN, method string) string {
	// facadeSymbolID:
	//   "scip-php composer laravel/framework <ver> Illuminate/Support/Facades/Auth#user()."
	// We want:
	//   "scip-php composer laravel/framework <ver> Illuminate/Auth/AuthManager#user()."
	parts := strings.SplitN(facadeSymbolID, " ", 5)
	if len(parts) < 5 {
		return ""
	}
	return strings.Join(parts[:4], " ") + " " + backingFQN + "#" + method + "()."
}
