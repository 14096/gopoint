# gopoint

[![CI](https://github.com/14096/gopoint/actions/workflows/ci.yml/badge.svg)](https://github.com/14096/gopoint/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/14096/gopoint.svg)](https://pkg.go.dev/github.com/14096/gopoint)

A Go SDK for SharePoint REST and Microsoft Graph.

> **Status:** pre-release (`v0.0.1-beta1`). This project is not production ready.

## Installation

Requires Go 1.27 or newer.

```sh
go get github.com/14096/gopoint@v0.0.1-beta1
```

## Quick start

```go
import (
	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/graph"
	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint"
)

cred, err := auth.NewCertificateFromFile(tenantID, clientID, "app.pfx", "pw", auth.Options{})
if err != nil {
	log.Fatal(err)
}

sp, err := sharepoint.New("https://contoso.sharepoint.com/sites/team", cred, sharepoint.Options{})
if err != nil {
	log.Fatal(err)
}
lists, err := sp.Lists().All(ctx, httpx.ODataQuery{})

g, err := graph.New(cred, graph.Options{})
if err != nil {
	log.Fatal(err)
}
site, err := g.Sites().ByPath(ctx, "contoso.sharepoint.com", "/sites/team")
```

Runnable programs for every credential type are in [examples/](examples).

## Anything not wrapped yet

Both clients expose `Raw()`, the generic client (auth, retry, error mapping and paging included):

```go
var out map[string]any
err := g.Raw().Get(ctx, "/sites/root/permissions", &out)
err = sp.Raw().Post(ctx, "web/ensureuser", map[string]string{"logonName": "i:0#.f|membership|a@b.c"}, &out)
```

## Authentication

Credentials implement `auth.Credential` and work with both `graph.Client` and `sharepoint.Client`;
each client asks for `<resource>/.default`, so one credential serves both services.

| Credential                                  | Kind      | Use when                    | Entra app registration                                                                                                        |
| ------------------------------------------- | --------- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `NewDeviceCode`                             | delegated | headless / SSH / no browser | public client, "Allow public client flows" = yes                                                                              |
| `NewInteractiveBrowser`                     | delegated | desktop tools               | public client, redirect URI `http://localhost` (mobile & desktop platform)                                                    |
| `NewCertificate` / `NewCertificateFromFile` | app-only  | services, automation        | upload the public cert; grant **application** permissions (e.g. `Sites.Selected`, `Sites.FullControl.All`) with admin consent |

- SharePoint app-only access requires a **certificate**; client secrets are rejected by SharePoint.
- Certificates may be PEM (cert + unencrypted RSA key) or PFX. Only RSA keys are supported.
- Delegated credentials keep the refresh token in memory only; nothing is written to disk.
- Use `auth.Options{Authority: ...}` for sovereign clouds.
- Any other token source can be plugged in by implementing `auth.Credential` (for example an adapter around `azidentity`).

## SharePoint REST

```go
sp, _ := sharepoint.New("https://contoso.sharepoint.com/sites/team", cred, sharepoint.Options{})
items, _ := sp.Items().All(ctx, spx.ByTitle("Projects"), httpx.ODataQuery{Filter: "Status eq 'Open'"})
```

| Accessor         | Package                   | Operations                                                  |
| ---------------- | ------------------------- | ----------------------------------------------------------- |
| `Web()`          | `sharepoint/web`          | get, subwebs, create subweb                                 |
| `Lists()`        | `sharepoint/lists`        | get, all, create, update, delete                            |
| `Fields()`       | `sharepoint/fields`       | list, get, create, update, delete (web or list)             |
| `ContentTypes()` | `sharepoint/contenttypes` | list, get, create, delete (web or list)                     |
| `Items()`        | `sharepoint/items`        | get, query (OData, paged), create, update, delete           |
| `Files()`        | `sharepoint/files`        | list files/folders, create folder, upload, download, delete |

Lists are addressed with `spx.ByTitle` / `spx.ByID`; operations valid on a web or a list take an
`spx.Scope` (`spx.Web` or a `ListRef`). Write calls automatically carry a cached `X-RequestDigest`.

## Microsoft Graph

Graph v1.0 by default; set `graph.Options.BaseURL` for `beta` or national clouds.

```go
g, _ := graph.New(cred, graph.Options{})
site, _ := g.Sites().ByPath(ctx, "contoso.sharepoint.com", "/sites/team")
for it, err := range g.Items().Iter(ctx, site.ID, "Projects", httpx.ODataQuery{Top: 100}) {
	if err != nil {
		return err
	}
	fmt.Println(it.ID)
}
```

| Accessor         | Package              | Operations                                                                                    |
| ---------------- | -------------------- | --------------------------------------------------------------------------------------------- |
| `Sites()`        | `graph/sites`        | root, get, by path, search, subsites                                                          |
| `Lists()`        | `graph/lists`        | iter, all, get, create, delete                                                                |
| `Columns()`      | `graph/columns`      | list (site or list), create, delete                                                           |
| `Items()`        | `graph/items`        | iter, all, get, create, update, delete (fields expanded by default)                           |
| `ContentTypes()` | `graph/contenttypes` | list (site or list), get                                                                      |
| `Drives()`       | `graph/drives`       | for site, get, children, get item (by id/path), download, upload (simple + resumable), delete |

## Documentation

The API reference is on [pkg.go.dev](https://pkg.go.dev/github.com/14096/gopoint).
