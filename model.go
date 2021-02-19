package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

//go:embed templates/*

var templateDir embed.FS

var (
	allTemplates *template.Template
)

const (
	defaultRef = "master"
)

// Package is the packages to be hosted on your site.
type Package struct {
	DisplayName       string `json:"display_name"`
	ParentDisplayName string `json:"parent_display_name,omitempty"`
	GitURL            string `json:"git_url"`
	GitRef            string `json:"git_ref"`
	Description       string `json:"description"`
	IsSubPath         bool   `json:"is_sub_path"`
	GitParentURL      string `json:"git_parent"`
}

// ErrorData contains error message to be specified for
type ErrorData struct {
	Reason string
	Status int
}

func newError(errMsg string, errStatus int) *ErrorData {
	return &ErrorData{Reason: errMsg, Status: errStatus}
}

func (e *ErrorData) renderErr(w http.ResponseWriter, logger *zerolog.Logger) {
	w.WriteHeader(e.Status)
	err := allTemplates.ExecuteTemplate(w, "errorPage", e)
	logger.Err(err).Msg("error while rendering error page")
	return
}

func (p *Package) validate() error {
	if p.DisplayName == "" {
		return errors.New("package name cannot be empty")
	}
	if _, err := url.Parse(p.GitURL); err != nil {
		return errors.New("git url has to be a valid url")
	}
	if p.GitRef == "" {
		p.GitRef = defaultRef
	}
	return nil
}

// GlobalConfig for this hosted site.
type GlobalConfig struct {
	GlobalDomain string          `json:"global_domain"`
	SiteTitle    string          `json:"site_title"`
	Packages     []Package       `json:"packages"`
	Logger       *zerolog.Logger `json:"-"`
}

func (gc *GlobalConfig) validate() error {
	if _, err := url.Parse(gc.GlobalDomain); err != nil {
		return errors.New("global domain has to be a valid url")
	}
	for _, p := range gc.Packages {
		if err := p.validate(); err != nil {
			return err
		}
	}
	return nil
}

func newGlobalConfig(jsonFilePath string, logger *zerolog.Logger) (*GlobalConfig, error) {
	abspath, err := filepath.Abs(jsonFilePath)
	if err != nil {
		return nil, err
	}
	if _, err = os.Stat(abspath); err != nil {
		return nil, err
	}
	var gconfig GlobalConfig
	fileContent, err := ioutil.ReadFile(abspath)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(fileContent, &gconfig); err != nil {
		return nil, err
	}
	gconfig.Logger = logger
	if err = gconfig.validate(); err != nil {
		return nil, err
	}

	var allFiles []string
	files, _ := templateDir.ReadDir("templates")
	for _, file := range files {
		filename := file.Name()
		if strings.HasSuffix(filename, ".html") {
			filePath := filepath.Join("templates", filename)
			allFiles = append(allFiles, filePath)
		}
	}
	funcMap := template.FuncMap{
		"getDomain": func(pkg string) string {
			return fmt.Sprintf("%s/%s", gconfig.GlobalDomain, pkg)
		},
	}
	allTemplates, err = template.New("").
		Funcs(funcMap).
		ParseFiles(allFiles...)

	return &gconfig, nil
}

func (gc *GlobalConfig) handleRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	// handle root endpoint
	if r.URL.Path == "/" {
		if err = allTemplates.ExecuteTemplate(w, "listPage", gc); err != nil {
			gc.Logger.Error().Msgf("%v", err)
			ed := newError("internal error", http.StatusInternalServerError)
			ed.renderErr(w, gc.Logger)
		}
		return
	}
	// handle health endpoint
	if r.URL.Path == "/health" {
		_, _ = w.Write([]byte("ok"))
		return
	}

	if r.URL.Path == "/favicon.svg" {
		b, err := templateDir.ReadFile("templates/favicon.svg")
		if err != nil {
			ed := newError("internal error", http.StatusInternalServerError)
			ed.renderErr(w, gc.Logger)
		}
		_, _ = w.Write(b)
		return
	}

	rPath := strings.Trim(r.URL.Path, "/")
	for _, pkg := range gc.Packages {
		if pkg.DisplayName == rPath {
			if err = allTemplates.ExecuteTemplate(w, "packagePage", pkg); err != nil {
				gc.Logger.Error().Msgf("%v", err)
				ed := newError("internal error", http.StatusInternalServerError)
				ed.renderErr(w, gc.Logger)
			}
			return
		}
	}
	// handle not found
	gc.Logger.Error().Msgf("%v", err)
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("not found"))
	return
}
