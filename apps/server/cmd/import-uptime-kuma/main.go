package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanyam-malik/pulsewatch/internal/importer/uptimekuma"

	"github.com/uptrace/bun/driver/sqliteshim"
)

func main() {
	var sourcePath, targetPath string
	var apply, includeSecrets, activate, publishPages bool
	flag.StringVar(&sourcePath, "source", "", "path to Uptime Kuma's data/kuma.db SQLite database")
	flag.StringVar(&targetPath, "target", "", "path to the existing Pulsewatch SQLite database")
	flag.BoolVar(&apply, "apply", false, "write the validated import (default is a read-only preview)")
	flag.BoolVar(&includeSecrets, "include-secrets", false, "explicitly import supported channel credentials; channels stay inactive and monitor secrets stay excluded")
	flag.BoolVar(&activate, "activate", false, "activate imported monitors that were active in Uptime Kuma")
	flag.BoolVar(&publishPages, "publish-pages", false, "publish imported pages that were published in Uptime Kuma")
	flag.Parse()

	if sourcePath == "" || targetPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	ctx := context.Background()
	source, err := openSQLite(sourcePath, true)
	if err != nil {
		fatal(err)
	}
	defer source.Close()
	target, err := openSQLite(targetPath, false)
	if err != nil {
		fatal(err)
	}
	defer target.Close()

	options := uptimekuma.Options{
		IncludeSecrets: includeSecrets,
		Activate:       activate,
		PublishPages:   publishPages,
	}
	plan, err := uptimekuma.Read(ctx, source, options)
	if err != nil {
		fatal(err)
	}
	if err := uptimekuma.ValidateTarget(ctx, target, plan); err != nil {
		fatal(err)
	}
	if apply {
		if err := uptimekuma.Apply(ctx, target, plan, options); err != nil {
			fatal(err)
		}
	}
	result := struct {
		Mode string `json:"mode"`
		uptimekuma.Summary
	}{Mode: "preview", Summary: plan.Summary}
	if apply {
		result.Mode = "applied"
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fatal(err)
	}
}

func openSQLite(path string, readOnly bool) (*sql.DB, error) {
	path = filepath.Clean(path)
	if path == "." {
		return nil, errors.New("database path must not be empty")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve SQLite path: %w", err)
	}
	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}
	mode := "rw"
	if readOnly {
		mode = "ro"
	}
	u.RawQuery = "mode=" + mode
	db, err := sql.Open(sqliteshim.ShimName, u.String())
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to SQLite database: %w", err)
	}
	return db, nil
}

func fatal(err error) {
	// Do not print SQL/config payloads or credentials alongside a failure.
	fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error()))
	os.Exit(1)
}
