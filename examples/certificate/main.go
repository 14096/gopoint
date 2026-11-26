// App-only certificate authentication; uploads a file to a document library.
//
//	TENANT_ID=... CLIENT_ID=... CERT_FILE=app.pfx CERT_PASSWORD=... \
//	SITE_URL=https://contoso.sharepoint.com/sites/team go run ./examples/certificate
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/14096/gopoint/auth"
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
	sp, err := sharepoint.New(os.Getenv("SITE_URL"), cred, sharepoint.Options{})
	if err != nil {
		log.Fatal(err)
	}
	f, err := sp.Files().Upload(ctx, "/sites/team/Shared Documents", "hello.txt", []byte("hello"), true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("uploaded", f.ServerRelativeURL)
}
