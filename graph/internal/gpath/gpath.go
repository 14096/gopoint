// Package gpath builds Graph resource paths shared by the graph packages.
package gpath

import "net/url"

// Item is the path of a drive item.
func Item(drive, id string) string {
	return "/drives/" + url.PathEscape(drive) + "/items/" + url.PathEscape(id)
}

// Site is the path of a site.
func Site(id string) string { return "/sites/" + url.PathEscape(id) }
