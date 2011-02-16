// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Cookie{} represents a parsed RFC 2109 "Set-Cookie" line in HTTP
// Response headers, extended with the HttpOnly attribute.
// Cookie{} is also used to represent parsed "Cookie" lines in
// HTTP Request headers. In this case, only the Value@ field is
// significant.
type Cookie struct {
	Value      string
	Path       string
	Domain     string
	Comment    string
	Version    string
	Expires    time.Time
	ExpiresRaw string
	MaxAge     int64	// Max age in nanoseconds
	Secure     bool
	HttpOnly   bool
	Raw        string
	Unparsed   []string	// Raw text of unparsed attribute-value pairs
}

// Cookies{} represents the collection of cookies found in a header.
// Individual cookies are represented by Cookie{} objects.
// For Request headers, only the Value@ field of Cookie{} is significant.
type Cookies map[string]Cookie

// ExtractSetCookies() parses all "Set-Cookie" values from
// the header h#, removes the successfully parsed values from the 
// "Set-Cookie" key in h# and returns the parsed Cookie{}s.
// TODO: Attribute values must be unescaped using the QUOTED-WORD convention.
func ExtractSetCookies(h map[string][]string) *Cookies {
	kk := new(Cookies)
	sk, ok := h["Set-Cookie"]
	if !ok {
		return kk
	}
	unparsed := make([]string, 0, 3)
	for _, ktext := range sk {
		parts := strings.Split(ktext, ";", -1)
		if len(parts) == 0 {
			unparsed = append(unparsed, ktext)
			continue
		}
		nv := strings.Split(strings.TrimSpace(parts[0]), "=", 2)		// Name=Value
		if len(nv) != 2 {
			unparsed = append(unparsed, ktext)
			continue
		}
		c := Cookie{ 
			Value:    nv[1], 
			MaxAge:   -1,	// Not specified
			Raw:      ktext,
			Unparsed: make([]string, 0, 1),
		}
		for i := 1; i < len(parts); i++ {
			av := strings.Split(strings.TrimSpace(parts[i]), "=", 2)	// Attribute=Value
			if len(av) == 1 {
				switch strings.ToLower(av[0]) {
				case "secure":
					c.Secure = true
				case "httponly":
					c.HttpOnly = true
				default:
					c.Unparsed = append(c.Unparsed, parts[i])
				}
			} else if len(av) == 2 {
				switch strings.ToLower(av[0]) {
				case "comment":
					c.Comment = av[1]
				case "domain":
					c.Domain = av[1]
				case "max-age":
					secs, err := strconv.Atoi64(av[1])
					if err != nil || secs < 0 {
						c.Unparsed = append(c.Unparsed, parts[i])
						continue
					}
					c.MaxAge = 1e9 * secs
				case "expires":
					c.ExpiresRaw = av[1]
					exptime, err := time.Parse(time.RFC1123, av[1])
					if err != nil {
						c.Unparsed = append(c.Unparsed, parts[i])
						continue
					}
					c.Expires = *exptime
				case "path":
					c.Path = av[1]
				case "secure":
					c.Secure = true
				case "version":
					c.Version = av[1]
				case "httponly":
					c.HttpOnly = true
				default:
					c.Unparsed = append(c.Unparsed, parts[i])
				}
			}
		} // Cookie attribute-value iteration
		(*kk)[nv[0]] = c
	} // header "Set-Cookie" value iteration
	if len(unparsed) > 0 {
		h["Set-Cookie"] = unparsed
	} else {
		h["Set-Cookie"] = nil, false
	}
	return kk
}

// ExtractCookies() parses all "Cookie" values from
// the header h#, removes the successfully parsed values from the 
// "Cookie" key in h# and returns the parsed Cookie{}s.
// TODO: Attribute values must be unescaped using the QUOTED-WORD convention.
func ExtractCookies(h map[string][]string) *Cookies {
	kk := new(Cookies)
	sk, ok := h["Cookie"]
	if !ok {
		return kk
	}
	unparsed := make([]string, 0, 3)
	??
	for _, ktext := range sk {
		parts := strings.Split(ktext, ";", -1)
		if len(parts) == 0 {
			unparsed = append(unparsed, ktext)
			continue
		}
		nv := strings.Split(strings.TrimSpace(parts[0]), "=", 2)	// Name=Value
		if len(nv) != 2 {
			unparsed = append(unparsed, ktext)
			continue
		}
		c := Cookie{ 
			Value:    nv[1], 
			MaxAge:   -1,	// Not specified
			Raw:      ktext,
			Unparsed: make([]string, 0, 1),
		}
		for i := 1; i < len(parts); i++ {
			av := strings.Split(strings.TrimSpace(parts[i]), "=", 2)	// Attribute=Value
			if len(av) == 1 {
				switch strings.ToLower(av[0]) {
				case "secure":
					c.Secure = true
				case "httponly":
					c.HttpOnly = true
				default:
					c.Unparsed = append(c.Unparsed, parts[i])
				}
			} else if len(av) == 2 {
				switch strings.ToLower(av[0]) {
				case "comment":
					c.Comment = av[1]
				case "domain":
					c.Domain = av[1]
				case "max-age":
					secs, err := strconv.Atoi64(av[1])
					if err != nil || secs < 0 {
						c.Unparsed = append(c.Unparsed, parts[i])
						continue
					}
					c.MaxAge = 1e9 * secs
				case "expires":
					c.ExpiresRaw = av[1]
					exptime, err := time.Parse(time.RFC1123, av[1])
					if err != nil {
						c.Unparsed = append(c.Unparsed, parts[i])
						continue
					}
					c.Expires = *exptime
				case "path":
					c.Path = av[1]
				case "secure":
					c.Secure = true
				case "version":
					c.Version = av[1]
				case "httponly":
					c.HttpOnly = true
				default:
					c.Unparsed = append(c.Unparsed, parts[i])
				}
			}
		} // Cookie attribute-value iteration
		(*kk)[nv[0]] = c
	} // header "Set-Cookie" value iteration
	if len(unparsed) > 0 {
		h["Set-Cookie"] = unparsed
	} else {
		h["Set-Cookie"] = nil, false
	}
	return kk
	??
}

func (kk *Cookie) WriteSetCookies(w io.Writer) os.Error {
	return nil
}

func (kk *Cookie) WriteCookies(w io.Writer) os.Error {
	return nil
}
