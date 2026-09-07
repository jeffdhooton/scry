// Run from the repository root to inspect unmodified indexer output.
package main

import (
 "encoding/json"
 "os"
 scip "github.com/scip-code/scip/bindings/go/scip"
 "google.golang.org/protobuf/proto"
)
func main() {
 b, err := os.ReadFile(os.Args[1]); if err != nil { panic(err) }
 var idx scip.Index
 if err := proto.Unmarshal(b, &idx); err != nil { panic(err) }
 enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  ")
 if err := enc.Encode(&idx); err != nil { panic(err) }
}
