// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import (
	"context"
	"net/url"
)

// Paginator walks every page of a list endpoint, following the response's
// links.next cursor. It is the value returned by CrudResource.Paginate and is
// the Go-idiom equivalent of Python's ReadResource.paginate()'s PaginatedIterator
// (signalwire/rest/_base.py + _pagination.py): it extracts resp[dataKey] as the
// page's items and follows resp["links"]["next"], carrying that URL's query
// params into the next fetch until no next link remains.
//
// It is driven by the namespaces HTTPClient interface (the same one every
// resource uses), so it lives in this package without importing the parent rest
// package (which would create an import cycle: rest already imports namespaces).
type Paginator struct {
	ctx     context.Context
	http    HTTPClient
	path    string
	params  map[string]string
	dataKey string
	done    bool
}

// NewPaginator builds a Paginator for path against client. dataKey is the JSON
// key holding the page's item array (defaults to "data"). params seeds the first
// request's query (nil is fine). ctx is threaded onto every page fetch so the
// whole walk is cancellable; a nil ctx falls back to context.Background().
func NewPaginator(ctx context.Context, client HTTPClient, path string, params map[string]string, dataKey string) *Paginator {
	if ctx == nil {
		ctx = context.Background()
	}
	if params == nil {
		params = map[string]string{}
	}
	if dataKey == "" {
		dataKey = "data"
	}
	return &Paginator{ctx: ctx, http: client, path: path, params: params, dataKey: dataKey}
}

// Next fetches the next page. It returns the page's items, hasMore (true when a
// links.next cursor was present so a further Next will fetch more), and any
// error. Once exhausted it returns (nil, false, nil) on every subsequent call —
// the Go-idiom equivalent of Python's StopIteration.
func (p *Paginator) Next() ([]map[string]any, bool, error) {
	if p.done {
		return nil, false, nil
	}

	resp, err := p.http.Get(p.ctx, p.path, p.params)
	if err != nil {
		return nil, false, err
	}

	var items []map[string]any
	if raw, ok := resp[p.dataKey]; ok {
		if arr, ok := raw.([]any); ok {
			for _, v := range arr {
				if m, ok := v.(map[string]any); ok {
					items = append(items, m)
				}
			}
		}
	}

	// Follow links.next when it is a non-empty URL and this page had items.
	if linksRaw, ok := resp["links"]; ok {
		if links, ok := linksRaw.(map[string]any); ok {
			if nextRaw, ok := links["next"]; ok {
				if nextURL, ok := nextRaw.(string); ok && nextURL != "" && len(items) > 0 {
					if parsed, perr := url.Parse(nextURL); perr == nil {
						next := map[string]string{}
						for k, v := range parsed.Query() {
							if len(v) > 0 {
								next[k] = v[0]
							}
						}
						p.params = next
						return items, true, nil
					}
				}
			}
		}
	}

	p.done = true
	return items, false, nil
}

// ForEach invokes fn for every item across all pages, fetching pages lazily via
// Next. It stops early (returning that error) if fn returns non-nil, and stops
// normally when no more pages remain.
func (p *Paginator) ForEach(fn func(map[string]any) error) error {
	for {
		items, hasMore, err := p.Next()
		if err != nil {
			return err
		}
		for _, item := range items {
			if ferr := fn(item); ferr != nil {
				return ferr
			}
		}
		if !hasMore {
			return nil
		}
	}
}
