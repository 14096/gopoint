// Interactive browser sign-in, then show the root site through Graph.
//
//	TENANT_ID=... CLIENT_ID=... go run ./examples/browser
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/graph"
)

func main() {
	cred, err := auth.NewInteractiveBrowser(auth.BrowserConfig{
		TenantID: os.Getenv("TENANT_ID"),
		ClientID: os.Getenv("CLIENT_ID"),
	})
	if err != nil {
		log.Fatal(err)
	}
	g, err := graph.New(cred, graph.Options{})
	if err != nil {
		log.Fatal(err)
	}
	site, err := g.Sites().Root(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(site.DisplayName, site.WebURL)
}
