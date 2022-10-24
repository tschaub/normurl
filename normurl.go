package normurl

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// Locator represents a file path or a URL.
type Locator struct {
	url  *url.URL
	file bool
}

type jsonLocator struct {
	Url  string
	File bool
}

var _ json.Unmarshaler = (*Locator)(nil)

// UnmarshalJSON creates a locator from JSON data
func (l *Locator) UnmarshalJSON(data []byte) error {
	var jl jsonLocator
	if err := json.Unmarshal(data, &jl); err != nil {
		return err
	}

	if jl.Url == "" {
		return fmt.Errorf("missing url")
	}

	nl, newErr := New(jl.Url)
	if newErr != nil {
		return newErr
	}

	if jl.File != nl.file {
		return fmt.Errorf("file flag mismatch")
	}

	l.file = jl.File
	l.url = nl.url

	return nil
}

var _ json.Marshaler = (*Locator)(nil)

// MarshalJSON encodes a locator as JSON
func (l *Locator) MarshalJSON() ([]byte, error) {
	jl := jsonLocator{
		Url:  l.url.String(),
		File: l.file,
	}
	return json.Marshal(jl)
}

func (l *Locator) String() string {
	return l.url.String()
}

// SetQueryParam updates the query param for a URL (pass an empty string to delete a param).
func (l *Locator) SetQueryParam(param string, value string) {
	if l.file {
		return
	}
	query := l.url.Query()
	if value != "" {
		query.Set(param, value)
	} else {
		query.Del(param)
	}
	l.url.RawQuery = query.Encode()
}

// IsFilepath checks if a locator is a file path.
func (l *Locator) IsFilepath() bool {
	return l.file
}

// New creates a locator.
func New(s string) (*Locator, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" {
		if !filepath.IsAbs(s) {
			return nil, fmt.Errorf("expected absolute path")
		}
		loc := &Locator{
			url:  u,
			file: true,
		}
		return loc, nil
	}

	if u.Scheme == "file" {
		path := u.Path
		if runtime.GOOS == "windows" {
			path = filepath.FromSlash(strings.TrimPrefix(path, "/"))
		}
		u.Scheme = ""
		u.Path = path
		loc := &Locator{
			url:  u,
			file: true,
		}
		return loc, nil
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %s", u.Scheme)
	}

	return &Locator{url: u}, nil
}

// Resolve creates a new locator from a base.
func (base *Locator) Resolve(s string) (*Locator, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "" {
		return New(s)
	}

	if base.file {
		if filepath.IsAbs(s) {
			loc := &Locator{
				url:  u,
				file: true,
			}
			return loc, nil
		}

		baseDir := filepath.Dir(base.url.Path)
		path := filepath.Join(baseDir, s)
		loc := &Locator{
			url:  &url.URL{Path: path},
			file: true,
		}
		return loc, nil
	}

	resolved := base.url.ResolveReference(u)
	loc := &Locator{
		url:  resolved,
		file: false,
	}
	return loc, nil
}
