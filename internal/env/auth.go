package env

import (
	"encoding/base64"
	"fmt"

	"github.com/MendezCarl/sailor.git/internal/request"
)

// applyAuth injects an Authorization (or custom) header derived from req.Auth
// into a copy of the request. If req.Auth is nil or Type is empty, the
// original request is returned unchanged.
//
// An explicit header in req.Headers always wins over the auth block.
// applyAuth must be called after variable interpolation.
func applyAuth(req *request.Request) (*request.Request, error) {
	if req.Auth == nil || req.Auth.Type == "" {
		return req, nil
	}

	out := *req
	if out.Headers == nil {
		out.Headers = make(map[string]string)
	}

	switch req.Auth.Type {
	case "bearer":
		if req.Auth.Token == "" {
			return nil, fmt.Errorf("auth.token is required for bearer auth")
		}
		if _, exists := out.Headers["Authorization"]; !exists {
			out.Headers["Authorization"] = "Bearer " + req.Auth.Token
		}

	case "basic":
		if req.Auth.Username == "" {
			return nil, fmt.Errorf("auth.username is required for basic auth")
		}
		encoded := base64.StdEncoding.EncodeToString(
			[]byte(req.Auth.Username + ":" + req.Auth.Password),
		)
		if _, exists := out.Headers["Authorization"]; !exists {
			out.Headers["Authorization"] = "Basic " + encoded
		}

	case "apikey":
		if req.Auth.Key == "" {
			return nil, fmt.Errorf("auth.key is required for apikey auth")
		}
		h := req.Auth.Header
		if h == "" {
			h = "Authorization"
		}
		if _, exists := out.Headers[h]; !exists {
			out.Headers[h] = req.Auth.Key
		}

	default:
		return nil, fmt.Errorf("auth type %q is not supported; valid values: bearer, basic, apikey", req.Auth.Type)
	}

	return &out, nil
}
