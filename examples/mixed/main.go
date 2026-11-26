// One credential used for both Graph and SharePoint REST, plus the generic
// client for an endpoint that has no typed wrapper.
//
//	TENANT_ID=... CLIENT_ID=... CERT_FILE=app.pfx CERT_PASSWORD=... \
//	SITE_URL=https://contoso.sharepoint.com/sites/team go run ./examples/mixed
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/graph"
	"github.com/14096/gopoint/sharepoint"
)

func main() {
	ctx := context.Background()
	cred, err := auth.NewCertificateFromFile(
		os.Getenv("TENANT_ID"), os.Getenv("CLIENT_ID"),
		os.Getenv("CERT_FILE"), os.Getenv("CERT_PASSWORD"), auth.Options{})
	if err != nil {
		log.Fatal(err)
	}
	siteURL := os.Getenv("SITE_URL")
	u, err := url.Parse(siteURL)
	if err != nil {
		log.Fatal(err)
	}

	g, err := graph.New(cred, graph.Options{})
	if err != nil {
		log.Fatal(err)
	}
	site, err := g.Sites().ByPath(ctx, u.Host, u.Path)
	if err != nil {
		log.Fatal(err)
	}

	sp, err := sharepoint.New(siteURL, cred, sharepoint.Options{})
	if err != nil {
		log.Fatal(err)
	}
	w, err := sp.Web().Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("graph:", site.ID, "| sharepoint:", w.Title)

	// Not wrapped yet: use the generic client.
	var perms map[string]any
	if err := g.Raw().Get(ctx, "/sites/"+site.ID+"/permissions", &perms); err != nil {
		log.Fatal(err)
	}
	fmt.Println(perms)
}
