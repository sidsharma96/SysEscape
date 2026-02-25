// migrate — database migration runner for Postgres schema management.
// Thin CLI wrapper around golang-migrate.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const defaultDatabaseURL = "postgres://ser:ser@localhost:5432/ser?sslmode=disable"
const migrationsPath = "file://migrations"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "up":
		runUp()
	case "down":
		runDown()
	case "create":
		runCreate()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: migrate <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  up              Apply all pending migrations")
	fmt.Fprintln(os.Stderr, "  down N          Roll back N migrations")
	fmt.Fprintln(os.Stderr, "  create NAME     Create new .up.sql and .down.sql files")
}

func databaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return defaultDatabaseURL
}

func newMigrate() (*migrate.Migrate, error) {
	return migrate.New(migrationsPath, databaseURL())
}

func runUp() {
	m, err := newMigrate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize migrate: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			fmt.Println("No new migrations to apply.")
			return
		}
		fmt.Fprintf(os.Stderr, "Migration up failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migrations applied successfully.")
}

func runDown() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: migrate down N")
		os.Exit(1)
	}

	n, err := strconv.Atoi(os.Args[2])
	if err != nil || n < 1 {
		fmt.Fprintln(os.Stderr, "N must be a positive integer")
		os.Exit(1)
	}

	m, err := newMigrate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize migrate: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	err = m.Steps(-n)
	if err != nil {
		if err == migrate.ErrNoChange {
			fmt.Println("No migrations to roll back.")
			return
		}
		fmt.Fprintf(os.Stderr, "Migration down failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Rolled back %d migration(s) successfully.\n", n)
}

func runCreate() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: migrate create NAME")
		os.Exit(1)
	}

	name := os.Args[2]

	// Read existing migration files to determine next sequence number.
	entries, err := os.ReadDir("migrations")
	if err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll("migrations", 0o755); mkErr != nil {
				fmt.Fprintf(os.Stderr, "Failed to create migrations directory: %v\n", mkErr)
				os.Exit(1)
			}
			entries = nil
		} else {
			fmt.Fprintf(os.Stderr, "Failed to read migrations directory: %v\n", err)
			os.Exit(1)
		}
	}

	next := 1
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if len(n) >= 4 {
			if num, err := strconv.Atoi(n[:4]); err == nil && num >= next {
				next = num + 1
			}
		}
	}

	upFile := fmt.Sprintf("migrations/%04d_%s.up.sql", next, name)
	downFile := fmt.Sprintf("migrations/%04d_%s.down.sql", next, name)

	if err := os.WriteFile(upFile, []byte("-- "+upFile+"\n"), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create %s: %v\n", upFile, err)
		os.Exit(1)
	}
	if err := os.WriteFile(downFile, []byte("-- "+downFile+"\n"), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create %s: %v\n", downFile, err)
		os.Exit(1)
	}

	fmt.Printf("Created migration files:\n  %s\n  %s\n", upFile, downFile)
}
