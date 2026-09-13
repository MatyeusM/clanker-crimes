// Command warehouse-server is the single-binary entrypoint (PLAN.md §9).
//
// Usage: warehouse-server [-listen ADDR] [-db PATH] [serve|migrate|version|backup DEST]
//
// With no subcommand (or `serve`) it migrates then serves. Config comes
// from the environment (WAREHOUSE_LISTEN, WAREHOUSE_DATABASE,
// WAREHOUSE_LOG_LEVEL); when an env var is set it wins over the
// matching flag, so flags are defaults for unset env vars only.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"warehouse-server/internal/config"
	"warehouse-server/internal/db"
	"warehouse-server/internal/httpapi"
)

// version is overridden at build time: go build -ldflags "-X main.version=x.y.z".
var version = "dev"

// serverStartTime records when the process started for uptime reporting.
var serverStartTime time.Time

// init captures the process start time as early as possible so that
// uptime measurements cover initialization as well as serving.
func init() {
	serverStartTime = time.Now()
}

// uptime returns how long the process has been running.
func uptime() time.Duration { return time.Since(serverStartTime) }

// logStartupBanner logs an ASCII startup banner with the effective
// runtime configuration so that deployments can be eyeballed in logs.
func logStartupBanner(listen, database string) {
	log.Printf(`
 __        __   _    ____  _____ _   _  ___  _   _ ____  _____
 \ \      / /  / \  |  _ \| ____| | | |/ _ \| | | | ___|| ____|
  \ \ /\ / /  / _ \ | |_) |  _| | |_| | | | | | | |___ \|  _|
   \ V  V /  / ___ \|  _ <| |___|  _  | |_| | |_| |___) | |___
    \_/\_/  /_/   \_\_| \_\_____|_| |_|\___/ \___/|____/|_____|
  listen=%s database=%s version=%s uptime=%s`,
		listen, database, version, uptime().Round(time.Millisecond))
}

// openDatabase opens the SQLite file from cfg and applies pending
// migrations, returning the ready pool.
func openDatabase(cfg config.Config) (*sql.DB, error) {
	conn, err := db.Open(cfg.Database)
	if err != nil {
		return nil, err
	}
	version, err := db.Migrate(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	log.Printf("database %s at schema version %d", cfg.Database, version)
	return conn, nil
}

func run(args []string) int {
	fs := flag.NewFlagSet("warehouse-server", flag.ContinueOnError)
	listenFlag := fs.String("listen", config.DefaultListen, "HTTP listen address")
	dbFlag := fs.String("db", config.DefaultDatabase, "SQLite database file")
	// Accept the subcommand anywhere among the flags: pull the first
	// bare word matching a known subcommand so both
	// `warehouse-server serve -db x` and `warehouse-server -db x serve` work.
	sub, filtered := splitSub(args[1:])
	if err := fs.Parse(filtered); err != nil {
		return 2
	}
	pos := fs.Args()
	// A bare word that is not a known subcommand is a usage error,
	// never the serve path: otherwise `warehouse-server bogus`
	// would bind the listen port instead of failing fast.
	if sub == "" && len(pos) > 0 {
		log.Printf("unknown subcommand %q (want serve|migrate|version|backup)", pos[0])
		return 2
	}
	if sub == "version" {
		if len(pos) != 0 {
			log.Printf("usage: warehouse-server version")
			return 2
		}
		fmt.Println(version)
		return 0
	}
	cfg := resolveConfig(*listenFlag, *dbFlag)
	log.Printf("warehouse-server %s log_level=%s", version, cfg.LogLevel)
	switch sub {
	case "", "serve":
		if len(pos) != 0 {
			log.Printf("usage: warehouse-server [serve]")
			return 2
		}
		// Serve below.
	case "migrate":
		if len(pos) != 0 {
			log.Printf("usage: warehouse-server migrate")
			return 2
		}
		conn, err := openDatabase(cfg)
		if err != nil {
			log.Printf("migrate error: %v", err)
			return 1
		}
		_ = conn.Close()
		return 0
	case "backup":
		if len(pos) != 1 {
			log.Printf("usage: warehouse-server backup <dest.db>")
			return 2
		}
		dest := pos[0]
		conn, err := openDatabase(cfg)
		if err != nil {
			log.Printf("database error: %v", err)
			return 1
		}
		defer conn.Close()
		if err := db.Backup(conn, dest); err != nil {
			log.Printf("backup error: %v", err)
			return 1
		}
		log.Printf("backup wrote %s", dest)
		return 0
	default:
		// Unreachable: splitSub only returns known subcommands.
		log.Printf("unknown subcommand %q (want serve|migrate|version|backup)", sub)
		return 2
	}
	conn, err := openDatabase(cfg)
	if err != nil {
		log.Printf("database error: %v", err)
		return 1
	}
	defer conn.Close()
	api := httpapi.New(conn, version)
	logStartupBanner(cfg.Listen, cfg.Database)
	log.Printf("warehouse-server %s listening on %s database=%s", version, cfg.Listen, cfg.Database)
	if err := http.ListenAndServe(cfg.Listen, withRequestLog(api.Handler())); err != nil {
		log.Printf("server error: %v", err)
		return 1
	}
	return 0
}

// splitSub extracts the first bare word matching a known subcommand.
// Everything else (flags, values, backup dest) is returned for the
// flag set to parse.
func splitSub(argv []string) (string, []string) {
	sub := ""
	kept := make([]string, 0, len(argv))
	for _, a := range argv {
		if sub == "" && !isFlag(a) {
			switch a {
			case "serve", "migrate", "version", "backup":
				sub = a
				continue
			}
		}
		kept = append(kept, a)
	}
	return sub, kept
}

func isFlag(s string) bool { return len(s) > 0 && s[0] == '-' }

// resolveConfig overlays flags with the environment: a set env var
// wins (PLAN.md §9), otherwise the flag value is used.
func resolveConfig(listenFlag, dbFlag string) config.Config {
	cfg := config.Load()
	if v := os.Getenv("WAREHOUSE_LISTEN"); v == "" {
		cfg.Listen = listenFlag
	}
	if v := os.Getenv("WAREHOUSE_DATABASE"); v == "" {
		cfg.Database = dbFlag
	}
	return cfg
}

// withRequestLog logs one line per request (method, path, status).
// Minimal v1 observability: no sampling, no body logging.
func withRequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d", r.Method, r.URL.Path, rec.status)
	})
}

// statusRecorder captures the response status for access logs.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func main() {
	os.Exit(run(os.Args))
}
