package ddevapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: create required CI4 files in a temp dir
func makeCI4Layout(t *testing.T, root string) {
	t.Helper()
	// spark
	if err := os.WriteFile(filepath.Join(root, "spark"), []byte("#!/usr/bin/env php"), 0o644); err != nil {
		t.Fatalf("write spark: %v", err)
	}
	// app/Config/App.php
	appCfg := filepath.Join(root, "app", "Config")
	if err := os.MkdirAll(appCfg, 0o755); err != nil {
		t.Fatalf("mkdir App.php dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appCfg, "App.php"), []byte("<?php // stub"), 0o644); err != nil {
		t.Fatalf("write App.php: %v", err)
	}
	// public/index.php
	pub := filepath.Join(root, "public")
	if err := os.MkdirAll(pub, 0o755); err != nil {
		t.Fatalf("mkdir public: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pub, "index.php"), []byte("<?php // stub"), 0o644); err != nil {
		t.Fatalf("write public/index.php: %v", err)
	}
}

func TestIsCodeIgniterApp_DetectsTrue(t *testing.T) {
	root := t.TempDir()
	makeCI4Layout(t, root)
	app := &DdevApp{AppRoot: root}

	if !isCodeIgniterApp(app) {
		t.Fatalf("expected isCodeIgniterApp=true")
	}
}

func TestIsCodeIgniterApp_DetectsFalseWhenMissingFiles(t *testing.T) {
	root := t.TempDir()
	// only one file present
	if err := os.WriteFile(filepath.Join(root, "spark"), []byte("stub"), 0o644); err != nil {
		t.Fatalf("write spark: %v", err)
	}
	app := &DdevApp{AppRoot: root}

	if isCodeIgniterApp(app) {
		t.Fatalf("expected isCodeIgniterApp=false when files are missing")
	}
}

func TestBuildCodeIgniterDBConfig_MySQL(t *testing.T) {
	app := &DdevApp{}
	app.Database.Type = "mysql"
	got := buildCodeIgniterDBConfig(app)
	if !strings.Contains(got, "DBDriver = MySQLi") || !strings.Contains(got, "port = 3306") {
		t.Fatalf("mysql config not rendered correctly: %s", got)
	}
}

func TestBuildCodeIgniterDBConfig_MariaDB(t *testing.T) {
	app := &DdevApp{}
	app.Database.Type = "mariadb"
	got := buildCodeIgniterDBConfig(app)
	if !strings.Contains(got, "DBDriver = MySQLi") || !strings.Contains(got, "port = 3306") {
		t.Fatalf("mariadb config not rendered correctly: %s", got)
	}
}

func TestBuildCodeIgniterDBConfig_Postgres(t *testing.T) {
	app := &DdevApp{}
	app.Database.Type = "postgres"
	got := buildCodeIgniterDBConfig(app)
	if !strings.Contains(got, "DBDriver = Postgre") || !strings.Contains(got, "port = 5432") {
		t.Fatalf("postgres config not rendered correctly: %s", got)
	}
}

func TestBuildCodeIgniterDBConfig_UnknownTypeFallsBack(t *testing.T) {
	app := &DdevApp{}
	app.Database.Type = "sqlite"
	got := buildCodeIgniterDBConfig(app)
	if !strings.Contains(got, "unknown DDEV database type: sqlite") {
		t.Fatalf("should annotate unknown type, got: %s", got)
	}
	if !strings.Contains(got, "DBDriver = MySQLi") || !strings.Contains(got, "port = 3306") {
		t.Fatalf("unknown should fall back to mysql defaults, got: %s", got)
	}
}

func TestBuildCodeIgniterDBConfig_OmitDB(t *testing.T) {
	app := &DdevApp{OmitContainers: []string{"db"}}
	got := buildCodeIgniterDBConfig(app)
	if !strings.HasPrefix(strings.TrimSpace(got), "# Database omitted") {
		t.Fatalf("expected omission comment, got: %q", got)
	}
}

func TestIsDBOmitted(t *testing.T) {
	if isDBOmitted(&DdevApp{OmitContainers: []string{"db"}}) != true {
		t.Fatalf("omit_containers should be detected")
	}
	if isDBOmitted(&DdevApp{OmitContainers: []string{"web"}}) != false {
		t.Fatalf("db not omitted should be false")
	}
}

func TestCreateCodeIgniterSettingsFile(t *testing.T) {
	root := t.TempDir()
	makeCI4Layout(t, root)
	app := &DdevApp{AppRoot: root}
	app.Database.Type = "postgres"

	// provide an "env" template that should be copied
	if err := os.WriteFile(filepath.Join(root, "env"), []byte("# example\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	path, err := createCodeIgniterSettingsFile(app)
	if err != nil {
		t.Fatalf("create .env: %v", err)
	}
	if path != filepath.Join(root, ".env") {
		t.Fatalf("unexpected env path: %s", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	str := string(content)
	if !strings.Contains(str, "database.default.DBDriver = Postgre") {
		t.Fatalf("db stanza not written for postgres: %s", str)
	}
	if !strings.Contains(str, "app.baseURL") {
		t.Fatalf("baseURL not appended: %s", str)
	}

	// call again: should not append again because file exists
	before := len(str)
	if _, err := createCodeIgniterSettingsFile(app); err != nil {
		t.Fatalf("second call should not error: %v", err)
	}
	afterBytes, _ := os.ReadFile(path)
	if len(afterBytes) != before {
		t.Fatalf(".env should be unchanged on second call")
	}
}

func TestSetPathsAndUploadDirs(t *testing.T) {
	root := t.TempDir()
	app := &DdevApp{AppRoot: root}
	setCodeIgniterSiteSettingsPaths(app)
	if app.SiteSettingsPath != filepath.Join(root, ".env") {
		t.Fatalf("SiteSettingsPath wrong: %s", app.SiteSettingsPath)
	}
	if app.SiteDdevSettingsFile != "" {
		t.Fatalf("SiteDdevSettingsFile should be empty for CI4")
	}
	dirs := getCodeIgniterUploadDirs(app)
	if len(dirs) != 1 || dirs[0] != "writable/uploads" {
		t.Fatalf("unexpected upload dirs: %#v", dirs)
	}
}
