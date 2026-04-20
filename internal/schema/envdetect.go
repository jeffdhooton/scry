package schema

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectDSNFromEnv reads .env in the project directory and tries to construct
// a DSN from common environment variable conventions.
func DetectDSNFromEnv(projectDir string) (string, error) {
	envPath := filepath.Join(projectDir, ".env")
	vars, err := parseEnvFile(envPath)
	if err != nil {
		return "", fmt.Errorf("read .env: %w", err)
	}

	if url := vars["DATABASE_URL"]; url != "" {
		return url, nil
	}
	if dsn := vars["DATABASE_DSN"]; dsn != "" {
		return dsn, nil
	}

	if conn := vars["DB_CONNECTION"]; conn != "" {
		host := vars["DB_HOST"]
		if host == "" {
			host = "127.0.0.1"
		}
		port := vars["DB_PORT"]
		db := vars["DB_DATABASE"]
		user := vars["DB_USERNAME"]
		pass := vars["DB_PASSWORD"]

		if db == "" {
			return "", fmt.Errorf("DB_DATABASE is required in .env when using DB_CONNECTION")
		}

		switch conn {
		case "mysql":
			if port == "" {
				port = "3306"
			}
			return fmt.Sprintf("mysql://%s:%s@%s:%s/%s", user, pass, host, port, db), nil
		case "pgsql", "postgres", "postgresql":
			if port == "" {
				port = "5432"
			}
			return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, db), nil
		default:
			return "", fmt.Errorf("unsupported DB_CONNECTION type: %s", conn)
		}
	}

	return "", fmt.Errorf("no database DSN found in %s — expected DATABASE_URL, DATABASE_DSN, or DB_CONNECTION+DB_HOST+DB_DATABASE", envPath)
}

func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vars := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		vars[key] = value
	}
	return vars, scanner.Err()
}
