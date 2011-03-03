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

// TODO(petar): Explicitly forbid parsing of Set-Cookie attributes
// starting with '$', which have been used to hack into broken
// servers using the eventual Request headers containing those
// invalid attributes that may overwrite intended $Version, $Path, 
// etc. attributes.

// Cookie represents a parsed RFC 2965 "Set-Cookie" line in HTTP
// Response headers, extended with the HttpOnly attribute.
// Cookie is also used to represent parsed "Cookie" lines in
// HTTP Request headers.
type Cookie struct {
	Name    string
	Value   string
	Path    string
	Domain  string
	Comment string

	// Cookie versions 1 and 2 are defined in RFC 2965.
	// Read methods assign these values if they are explicitly 
	// seen while parsing, or use Version=0 otherwise. 
	// Write methods do not explicitly write the Version 
	// attribute if lower than 2, for compatibility reasons.
	Version    uint
	Expires    time.Time
	RawExpires string
	MaxAge     int64 // Max age in nanoseconds
	Secure     bool
	HttpOnly   bool
	Raw        string
	Unparsed   []string // Raw text of unparsed attribute-value pairs
}

// readSetCookies parses all "Set-Cookie" values from
// the header h, removes the successfully parsed values from the 
// "Set-Cookie" key in h and returns the parsed Cookies.
func readSetCookies(h Header) []*Cookie {
	cookies := []*Cookie{}
	var unparsedLines []string
	for _, line := range h["Set-Cookie"] {
		parts := strings.Split(strings.TrimSpace(line), ";", -1)
		if len(parts) == 1 && parts[0] == "" {
			continue
		}
		parts[0] = strings.TrimSpace(parts[0])
		j := strings.Index(parts[0], "=")
		if j < 0 {
			unparsedLines = append(unparsedLines, line)
			continue
		}
		name, value := parts[0][:j], parts[0][j+1:]
		value, err := URLUnescape(value)
		if err != nil {
			unparsedLines = append(unparsedLines, line)
			continue
		}
		c := &Cookie{
			Name:     name,
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

			attr, val := parts[i], ""
			if j := strings.Index(attr, "="); j >= 0 {
				attr, val = attr[:j], attr[j+1:]
				val, err = URLUnescape(val)
				if err != nil {
					c.Unparsed = append(c.Unparsed, parts[i])
					continue
				}
			}
			switch strings.ToLower(attr) {
			case "secure":
				c.Secure = true
				continue
			case "httponly":
				c.HttpOnly = true
				continue
			case "comment":
				c.Comment = val
				continue
			case "domain":
				c.Domain = val
				// TODO: Add domain parsing
				continue
			case "max-age":
				secs, err := strconv.Atoi64(val)
				if err != nil || secs < 0 {
					break
				}
				c.MaxAge = 1e9 * secs
				continue
			case "expires":
				c.RawExpires = val
				exptime, err := time.Parse(time.RFC1123, val)
				if err != nil {
					c.Expires = time.Time{}
					break
				}
				c.Expires = *exptime
				continue
			case "path":
				c.Path = val
				// TODO: Add path parsing
				continue
			case "version":
				c.Version, err = strconv.Atoui(val)
				if err != nil {
					c.Version = 0
					break
				}
				continue
			}
			c.Unparsed = append(c.Unparsed, parts[i])
		}
		cookies = append(cookies, c)
	}
	h["Set-Cookie"] = unparsedLines, unparsedLines != nil
	return cookies
}

// writeSetCookies writes the wire representation of the set-cookies
// to w. Each cookie is written on a separate "Set-Cookie: " line.
// This choice is made because HTTP parsers tend to have a limit on
// line-length, so it seems safer to place cookies on separate lines.
func writeSetCookies(kk []*Cookie, w io.Writer) os.Error {
	if kk == nil {
		return nil
	}
	lines := make([]string, 0, len(kk))
	for _, c := range kk {
		var value string = CanonicalHeaderKey(c.Name) + "=" + URLEscape(c.Value) + "; "
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

// readCookies parses all "Cookie" values from
// the header h, removes the successfully parsed values from the 
// "Cookie" key in h and returns the parsed Cookies.
func readCookies(h Header) []*Cookie {
	cookies := []*Cookie{}
	lines, ok := h["Cookie"]
	if !ok {
		return cookies
	}
	unparsedLines := []string{}
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), ";", -1)
		if len(parts) == 1 && parts[0] == "" {
			continue
		}
		// Per-line attributes
		var lineCookies = make(map[string]string)
		var version uint
		var path string
		var domain string
		var comment string
		var httponly bool
		for i := 0; i < len(parts); i++ {
			parts[i] = strings.TrimSpace(parts[i])
			if len(parts[i]) == 0 {
				continue
			}
			attr, val := parts[i], ""
			var err os.Error
			if j := strings.Index(attr, "="); j >= 0 {
				attr, val = attr[:j], attr[j+1:]
				val, err = URLUnescape(val)
				if err != nil {
					continue
				}
			}
			switch strings.ToLower(attr) {
			case "$httponly":
				httponly = true
			case "$version":
				version, err = strconv.Atoui(val)
				if err != nil {
					version = 0
					continue
				}
			case "$domain":
				domain = val
				// TODO: Add domain parsing
			case "$path":
				path = val
				// TODO: Add path parsing
			case "$comment":
				comment = val
			default:
				lineCookies[attr] = val
			}
		}
		if len(lineCookies) == 0 {
			unparsedLines = append(unparsedLines, line)
		}
		for n, v := range lineCookies {
			cookies = append(cookies, &Cookie{
				Name:     n,
				Value:    v,
				Path:     path,
				Domain:   domain,
				Comment:  comment,
				Version:  version,
				HttpOnly: httponly,
				MaxAge:   -1,
				Raw:      line,
			})
		}
	}
	h["Cookie"] = unparsedLines, len(unparsedLines) > 0
	return cookies
}

// writeCookies writes the wire representation of the cookies
// to w. Each cookie is written on a separate "Cookie: " line.
// This choice is made because HTTP parsers tend to have a limit on
// line-length, so it seems safer to place cookies on separate lines.
func writeCookies(kk []*Cookie, w io.Writer) os.Error {
	lines := make([]string, 0, len(kk))
	for _, c := range kk {
		n := c.Name
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
