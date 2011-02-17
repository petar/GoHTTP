// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textproto

// A MIMEHeader represents a MIME-style header mapping
// keys to sets of values.
type MIMEHeader map[string][]string

// Add adds the key, value pair to the header.
// It appends to any existing values associated with key.
func (h MIMEHeader) Add(key, value string) {
	key = CanonicalMIMEHeaderKey(key)

	oldValues, ok := h[key]
	if ok {
		oldValues[0] = oldValues[0] + "," + value
	} else {
		h[key] = []string{value}
	}
}

// Add adds the key, value pair to the header.
// It appends to any existing values associated with key.
func (h MIMEHeader) AddNewLine(key, value string) {
	key = CanonicalMIMEHeaderKey(key)

	oldValues, ok := h[key]
	if ok {
		h[key] = append(oldValues, value)
	} else {
		h[key] = []string{value}
	}
}

// Set sets the header entries associated with key to
// the single element value.  It replaces any existing
// values associated with key.
func (h MIMEHeader) Set(key, value string) {
	h[CanonicalMIMEHeaderKey(key)] = []string{value}
}

// Get gets the first value associated with the given key.
// If there are no values associated with the key, Get returns "".
// Get is a convenience method.  For more complex queries,
// access the map directly.
func (h MIMEHeader) Get(key string) string {
	if h == nil {
		return ""
	}
	v_, ok := h[CanonicalMIMEHeaderKey(key)]
	if !ok {
		return ""
	}
	return v_[0]
}

// Del deletes the values associated with key.
func (h MIMEHeader) Del(key string) {
	h[CanonicalMIMEHeaderKey(key)] = nil, false
}
