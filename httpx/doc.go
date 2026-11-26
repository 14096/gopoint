// Package httpx is the transport layer shared by the Graph and SharePoint
// clients. Client is also the generic client: it can call any REST endpoint
// that has no typed wrapper yet.
//
//	c, err := httpx.New(httpx.Config{BaseURL: "https://graph.microsoft.com/v1.0", Token: tokenFn})
//	var out map[string]any
//	err = c.Get(ctx, "/sites/root/permissions", &out)
package httpx
