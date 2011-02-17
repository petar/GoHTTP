// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"io"
	"os"
	"sort"
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
	Version    uint
	Expires    time.Time
	ExpiresRaw string
	MaxAge     int64 // Max age in nanoseconds
	Secure     bool
	HttpOnly   bool
	Raw        string
	Unparsed   []string // Raw text of unparsed attribute-value pairs
}

// Cookies{} represents the collection of cookies found in a header.
// Individual cookies are represented by Cookie{} objects.
// For Request headers, only the Value@ field of Cookie{} is significant.
type Cookies map[string]Cookie

// Get() returns the value of a cookie with the given key,
// or the empty string otherwise.
func (kk Cookies) Get(key string) string {
	if kk == nil {
		return ""
	}
	c, ok := kk[key]
	if !ok {
		return ""
	}
	return c.Value
}

// readSetCookies() parses all "Set-Cookie" values from
// the header h#, removes the successfully parsed values from the 
// "Set-Cookie" key in h# and returns the parsed Cookie{}s.
func readSetCookies(h map[string][]string) *Cookies {
	cookies := make(Cookies)
	lines, ok := h["Set-Cookie"]
	if !ok {
		return &cookies
	}
	unparsed_lines := make([]string, 0, 3)
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), ";", -1)
		if len(parts) == 1 && parts[0] == "" {
			continue
		}
		nv := strings.Split(strings.TrimSpace(parts[0]), "=", 2) // Name=Value
		if len(nv) != 2 {
			unparsed_lines = append(unparsed_lines, line)
			continue
		}
		name, err := URLUnescape(nv[0])
		if err != nil {
			unparsed_lines = append(unparsed_lines, line)
			continue
		}
		value, err := URLUnescape(nv[1])
		if err != nil {
			unparsed_lines = append(unparsed_lines, line)
			continue
		}
		c := Cookie{
			Value:    value,
			MaxAge:   -1, // Not specified
			Raw:      line,
			Unparsed: make([]string, 0, 1),
		}
		for i := 1; i < len(parts); i++ {
			parts[i] = strings.TrimSpace(parts[i])
			if len(parts[i]) == 0 {
				continue
			}
			av := strings.Split(parts[i], "=", 2) // Attribute=Value
			if len(av) == 1 {
				arg := av[0]
				switch strings.ToLower(arg) {
				case "secure":
					c.Secure = true
				case "httponly":
					c.HttpOnly = true
				default:
					c.Unparsed = append(c.Unparsed, parts[i])
				}
			} else if len(av) == 2 {
				arg0 := av[0]
				arg1, err := URLUnescape(av[1])
				if err != nil {
					c.Unparsed = append(c.Unparsed, parts[i])
					continue
				}
				switch strings.ToLower(arg0) {
				case "comment":
					c.Comment = arg1
				case "domain":
					c.Domain = arg1
					// TODO: Add domain parsing
				case "max-age":
					secs, err := strconv.Atoi64(arg1)
					if err != nil || secs < 0 {
						c.Unparsed = append(c.Unparsed, parts[i])
						continue
					}
					c.MaxAge = 1e9 * secs
				case "expires":
					c.ExpiresRaw = arg1
					exptime, err := time.Parse(time.RFC1123, arg1)
					if err != nil {
						c.Expires = time.Time{}
						c.Unparsed = append(c.Unparsed, parts[i])
						continue
					}
					c.Expires = *exptime
				case "path":
					c.Path = arg1
					// TODO: Add path parsing
				case "secure":
					c.Secure = true
				case "version":
					c.Version, err = strconv.Atoui(arg1)
					if err != nil {
						c.Version = 0
						c.Unparsed = append(c.Unparsed, parts[i])
						continue
					}
				case "httponly":
					c.HttpOnly = true
				default:
					c.Unparsed = append(c.Unparsed, parts[i])
				}
			}
		} // Cookie attribute-value iteration
		cookies[name] = c
	} // header "Set-Cookie" value iteration
	if len(unparsed_lines) > 0 {
		h["Set-Cookie"] = unparsed_lines
	} else {
		h["Set-Cookie"] = nil, false
	}
	return &cookies
}

// writeSetCookies() writes the wire representation of the set-cookies
// to w#. Each cookie is written on a separate "Set-Cookie: " line.
// This choice is made because HTTP parsers tend to have a limit on
// line-length, so it seems safer to place cookies on separate lines.
func (kk *Cookies) writeSetCookies(w io.Writer) os.Error {
	lines := make([]string, 0, len(*kk))
	for n, c := range *kk {
		var value string = CanonicalHeaderKey(n) + "=" + URLEscape(c.Value) + "; "
		var version string
		if c.Version > 1 {
			version = "Version=" + strconv.Uitoa(c.Version) + "; "
		}
		var path string
		if len(c.Path) > 0 {
			path = "Path=" + URLEscape(c.Path) + "; "
		}
		var domain string
		if len(c.Domain) > 0 {
			domain = "Domain=" + URLEscape(c.Domain) + "; "
		}
		var expires string
		if len(c.Expires.Zone) > 0 {
			expires = "Expires=" + c.Expires.Format(time.RFC1123) + "; "
		}
		var maxage string
		if c.MaxAge >= 0 {
			maxage = "Max-Age=" + strconv.Itoa64(c.MaxAge) + "; "
		}
		var httponly string
		if c.HttpOnly {
			httponly = "HttpOnly; "
		}
		var secure string
		if c.Secure {
			secure = "Secure; "
		}
		var comment string
		if len(c.Comment) > 0 {
			comment = "Comment=" + URLEscape(c.Comment) + "; "
		}
		lines = append(lines, "Set-Cookie: "+
			value+version+domain+path+expires+
			maxage+secure+httponly+comment+"\r\n")
	}
	sort.SortStrings(lines)
	for _, l := range lines {
		if _, err := io.WriteString(w, l); err != nil {
			return err
		}
	}
	return nil
}

// readCookies() parses all "Cookie" values from
// the header h#, removes the successfully parsed values from the 
// "Cookie" key in h# and returns the parsed Cookie{}s.
func readCookies(h map[string][]string) *Cookies {
	cookies := new(Cookies)
	lines, ok := h["Cookie"]
	if !ok {
		return cookies
	}
	var unparsed_lines []string = make([]string, 0, 3)
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), ";", -1)
		if len(parts) == 1 && parts[0] == "" {
			continue
		}
		// Per-line attributes
		var line_cookies = make(map[string]string)
		var version uint
		var path string
		var domain string
		var comment string
		var httponly bool
		for i := 1; i < len(parts); i++ {
			parts[i] = strings.TrimSpace(parts[i])
			if len(parts[i]) == 0 {
				continue
			}
			av := strings.Split(strings.TrimSpace(parts[i]), "=", 2) // Attribute=Value
			if len(av) == 1 {
				arg := av[0]
				switch strings.ToLower(arg) {
				case "$httponly":
					httponly = true
				}
			} else if len(av) == 2 {
				arg0 := av[0]
				arg1, err := URLUnescape(av[1])
				if err != nil {
					continue
				}
				switch strings.ToLower(arg0) {
				case "$version":
					version, err = strconv.Atoui(arg1)
					if err != nil {
						version = 0
						continue
					}
				case "$domain":
					domain = arg1
					// TODO: Add domain parsing
				case "$path":
					path = arg1
					// TODO: Add path parsing
				case "$comment":
					comment = arg1
				case "$httponly":
					httponly = true
				default:
					line_cookies[arg0] = arg1
				}
			}
		} // attribute-value iteration
		if len(line_cookies) == 0 {
			unparsed_lines = append(unparsed_lines, line)
		}
		for n, v := range line_cookies {
			(*cookies)[n] = Cookie{
				Value:    v,
				Path:     path,
				Domain:   domain,
				Comment:  comment,
				Version:  version,
				HttpOnly: httponly,
				Raw:      line,
			}
		}
	} // header "Cookie" line iteration

	if len(unparsed_lines) > 0 {
		h["Cookie"] = unparsed_lines
	} else {
		h["Cookie"] = nil, false
	}
	return cookies
}

// writeCookies() writes the wire representation of the cookies
// to w#. Each cookie is written on a separate "Cookie: " line.
// This choice is made because HTTP parsers tend to have a limit on
// line-length, so it seems safer to place cookies on separate lines.
func (kk *Cookies) writeCookies(w io.Writer) os.Error {
	lines := make([]string, 0, len(*kk))
	for n, c := range *kk {
		var value string = CanonicalHeaderKey(n) + "=" + URLEscape(c.Value) + "; "
		var version string
		if c.Version > 1 {
			version = "$Version=" + strconv.Uitoa(c.Version) + "; "
		}
		var path string
		if len(c.Path) > 0 {
			path = "$Path=" + URLEscape(c.Path) + "; "
		}
		var domain string
		if len(c.Domain) > 0 {
			domain = "$Domain=" + URLEscape(c.Domain) + "; "
		}
		var httponly string
		if c.HttpOnly {
			httponly = "$HttpOnly; "
		}
		var comment string
		if len(c.Comment) > 0 {
			comment = "$Comment=" + URLEscape(c.Comment) + "; "
		}
		lines = append(lines, "Cookie: "+
			version+value+domain+path+httponly+comment+"\r\n")
	}
	sort.SortStrings(lines)
	for _, l := range lines {
		if _, err := io.WriteString(w, l); err != nil {
			return err
		}
	}
	return nil
}
