package schema

import (
	"fmt"
	"strings"
)

// Introspector connects to a database and extracts its full schema.
type Introspector interface {
	Connect(dsn string) error
	Introspect() (*SchemaSnapshot, error)
	Close() error
}

// ParseDSN parses a DSN string and returns (dbType, driverDSN, error).
func ParseDSN(dsn string) (dbType string, driverDSN string, err error) {
	if strings.HasPrefix(dsn, "mysql://") {
		return "mysql", parseMySQLURI(dsn), nil
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return "postgres", dsn, nil
	}
	if strings.Contains(dsn, "tcp(") || strings.Contains(dsn, "@/") {
		return "mysql", dsn, nil
	}
	if strings.Contains(dsn, "host=") || strings.Contains(dsn, "sslmode=") {
		return "postgres", dsn, nil
	}
	return "", "", fmt.Errorf("cannot determine database type from DSN — use mysql:// or postgres:// prefix")
}

func parseMySQLURI(uri string) string {
	rest := strings.TrimPrefix(uri, "mysql://")

	var params string
	if idx := strings.Index(rest, "?"); idx >= 0 {
		params = rest[idx:]
		rest = rest[:idx]
	}

	var userInfo, hostPath string
	if idx := strings.LastIndex(rest, "@"); idx >= 0 {
		userInfo = rest[:idx]
		hostPath = rest[idx+1:]
	} else {
		hostPath = rest
	}

	var host, dbName string
	if idx := strings.Index(hostPath, "/"); idx >= 0 {
		host = hostPath[:idx]
		dbName = hostPath[idx+1:]
	} else {
		host = hostPath
	}

	if host != "" && !strings.Contains(host, ":") {
		host += ":3306"
	}

	result := ""
	if userInfo != "" {
		result = userInfo + "@"
	}
	if host != "" {
		result += "tcp(" + host + ")"
	}
	result += "/" + dbName + params

	return result
}
