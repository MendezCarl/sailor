package main

import (
	"github.com/MendezCarl/sailor.git/internal/config"
	"github.com/MendezCarl/sailor.git/internal/env"
	"github.com/MendezCarl/sailor.git/internal/request"
	"gopkg.in/yaml.v3"
)

// varFlags accumulates repeated --var key=value flags.
type varFlags []string

func (v *varFlags) String() string { return "" }
func (v *varFlags) Set(s string) error {
	*v = append(*v, s)
	return nil
}

// parseVarFlags converts a varFlags slice into an env.Vars map.
// Returns an error if any entry is not in key=value format.
func parseVarFlags(flags varFlags) (env.Vars, error) {
	vars := env.Vars{}
	for _, f := range flags {
		k, v, err := env.ParseVarFlag(f)
		if err != nil {
			return nil, err
		}
		vars[k] = v
	}
	return vars, nil
}

// resolveFollowRedirects returns the effective redirect preference.
// Priority: --no-follow-redirects > --follow-redirects > config value.
func resolveFollowRedirects(followFlag, noFollowFlag bool, cfg *config.Config) *bool {
	if noFollowFlag {
		v := false
		return &v
	}
	if followFlag {
		v := true
		return &v
	}
	return cfg.FollowRedirects
}

// requestToYAML converts a Request to YAML bytes suitable for stdout or file output.
// Uses a local type with omitempty so empty fields are not written.
func requestToYAML(req *request.Request) ([]byte, error) {
	type yamlReq struct {
		Name    string            `yaml:"name,omitempty"`
		Method  string            `yaml:"method"`
		URL     string            `yaml:"url"`
		Headers map[string]string `yaml:"headers,omitempty"`
		Params  map[string]string `yaml:"params,omitempty"`
		Body    string            `yaml:"body,omitempty"`
	}
	out := yamlReq{
		Name:    req.Name,
		Method:  req.Method,
		URL:     req.URL,
		Headers: req.Headers,
		Params:  req.Params,
		Body:    req.Body,
	}
	return yaml.Marshal(out)
}
