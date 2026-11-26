# gopoint

[![Go Reference](https://pkg.go.dev/badge/github.com/14096/gopoint.svg)](https://pkg.go.dev/github.com/14096/gopoint)

A Go SDK for SharePoint REST and Microsoft Graph.

> **Status:** pre-release (`v0.0.1-beta3`). This project is not production ready.

## Installation

Requires Go 1.27 or newer.

```sh
go get github.com/14096/gopoint@v0.0.1-beta3
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

Per-call options work on every `Raw()` helper, and generic helpers return typed results:

```go
site, err := httpx.Get[Site](ctx, g.Raw(), "/sites/root")
var item Item
resp, err := sp.Raw().Send(ctx, "GET", "web/lists/getbytitle('Tasks')/items(1)", nil, &item)
err = sp.Raw().Post(ctx, "web/lists/getbytitle('Tasks')/items(1)", fields, nil,
	httpx.WithHeader("X-HTTP-Method", "MERGE"), httpx.WithIfMatch(resp.ETag()))
```

Other escape hatches: `Raw().DoHTTP(ctx, *http.Request)`, `httpx.Config.Middleware` and `httpx.Config.ShouldRetry`
(not exposed through `Options`; build your own `httpx.Client` if you need them).

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

| Accessor         | Package                   | Operations                                                                                       |
| ---------------- | ------------------------- | ------------------------------------------------------------------------------------------------ |
| `Web()`          | `sharepoint/web`          | get, subwebs, create subweb                                                                      |
| `Lists()`        | `sharepoint/lists`        | get, all, create, update, delete                                                                 |
| `Fields()`       | `sharepoint/fields`       | list, get, create, update, delete (web or list)                                                  |
| `ContentTypes()` | `sharepoint/contenttypes` | list, get, create, delete (web or list)                                                          |
| `Items()`        | `sharepoint/items`        | get, query (OData, paged), create, update, delete                                                |
| `Files()`        | `sharepoint/files`        | list files/folders, create folder, upload, chunked upload (large files), download, delete        |
| `Pages()`        | `sharepoint/pages`        | modern site pages: list, get, create, update (draft), publish, delete                            |
| `Folders()`      | `sharepoint/folders`      | get, exists, children, create, rename, move, delete, recycle                                     |
| `Navigation()`   | `sharepoint/navigation`   | quick launch / top bar nodes: list, get, add, add child, update, delete                          |
| `Permissions()`  | `sharepoint/permissions`  | role assignments, grant/revoke, break/reset inheritance, effective permissions, role definitions |
| `RecycleBin()`   | `sharepoint/recyclebin`   | web / site bins: list, get, restore, delete, second stage, restore all, empty                    |
| `Users()`        | `sharepoint/users`        | site users, ensure user, site groups, group membership                                           |
| `Search()`       | `sharepoint/search`       | query (paged iterator), suggest                                                                  |
| `Sharing()`      | `sharepoint/sharing`      | create/delete sharing links, invite people                                                       |
| `Views()`        | `sharepoint/views`        | list views: all, get, by title, create, update, delete, view fields (add, remove, move, set)     |
| `HubSites()`     | `sharepoint/hubsites`     | list, get, current, register, unregister, update, join, leave                                    |
| `Webhooks()`     | `sharepoint/webhooks`     | list webhooks: list, add, renew, delete, plus an `http.Handler` for notifications                |

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

| Accessor          | Package               | Operations                                                          |
| ----------------- | --------------------- | ------------------------------------------------------------------- |
| `Sites()`         | `graph/sites`         | root, get, by path, search, subsites                                |
| `Lists()`         | `graph/lists`         | iter, all, get, create, delete                                      |
| `Columns()`       | `graph/columns`       | list (site or list), create, delete                                 |
| `Items()`         | `graph/items`         | iter, all, get, create, update, delete (fields expanded by default) |
| `ContentTypes()`  | `graph/contenttypes`  | list (site or list), get                                            |
| `Pages()`         | `graph/pages`         | site pages: iter, get, create, update, publish, delete              |
| `Folders()`       | `graph/folders`       | get, children, create, rename, move, copy, delete, permanent delete |
| `Permissions()`   | `graph/permissions`   | item permissions (list, get, update, delete), site app permissions  |
| `RecycleBin()`    | `graph/recyclebin`    | iter, restore, delete (**beta endpoint only**)                      |
| `Users()`         | `graph/users`         | me, users, groups, membership                                       |
| `Search()`        | `graph/search`        | Microsoft Search query (paged iterator)                             |
| `Sharing()`       | `graph/sharing`       | create link, invite, unshare, resolve sharing URL                   |
| `Drives()`        | `graph/drives`        | drives, children, download, upload (resumable), delete, delta       |
| `Subscriptions()` | `graph/subscriptions` | change notifications: create, get, iter, renew, delete, `Handler`   |
| `TermStore()`     | `graph/termstore`     | managed metadata: groups, sets, terms, create/delete term           |

## Batching

Both clients send many calls in one round trip, in input order. A failing sub-request does not fail the call; check `Err()` on each result.

```go
res, err := g.Batch(ctx, []graph.BatchRequest{{URL: "/sites/root"}, {URL: "/me"}}) // 20 per call
res, err = sp.Batch(ctx, []sharepoint.BatchRequest{{Path: "web?$select=Title"}})  // 100 per call
var web struct{ Title string }
err = res[0].JSON(&web)
```

## Delta queries

```go
changed, link, err := g.Drives().Delta(ctx, driveID, "") // empty link: full enumeration
// store link, later:
changed, link, err = g.Drives().Delta(ctx, driveID, link) // only changes; Item.Deleted marks removals
```

`g.Items().Delta(ctx, site, list, link)` does the same for list items.

## Large files

`sp.Files().UploadLarge(ctx, folderURL, name, readerAt, size, overwrite, progress)` uses SharePoint's chunked upload and keeps one chunk in memory.
`g.Drives().Upload` switches to a resumable upload session above 4 MB.

## Webhooks and notifications

```go
http.Handle("/hook", subscriptions.Handler("my-state", func(n subscriptions.Notification) { /* ... */ })) // Graph
http.Handle("/sp-hook", webhooks.Handler("my-state", func(n webhooks.Notification) { /* ... */ }))        // SharePoint
```

The handlers answer the validation handshake and drop notifications with a different `clientState`.
Subscriptions expire: renew them with `Renew` (Graph and SharePoint webhooks).

## Throttling

429, 503 and (for idempotent methods) 504 are retried with `Retry-After` and backoff. Set a decorated user agent,
which SharePoint Online treats better when throttling: `sharepoint.Options{UserAgent: httpx.ISVUserAgent("Acme", "SyncTool", "1.0")}`.
`Response.RateLimit()` exposes the `RateLimit-*` headers.
