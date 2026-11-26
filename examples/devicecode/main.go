// Device code sign-in, then list the lists of a SharePoint site via REST.
//
//	TENANT_ID=... CLIENT_ID=... SITE_URL=https://contoso.sharepoint.com/sites/team go run ./examples/devicecode
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint"
)

func main() {
	ctx := context.Background()
	cred, err := auth.NewDeviceCode(auth.DeviceCodeConfig{
		TenantID: os.Getenv("TENANT_ID"),
		ClientID: os.Getenv("CLIENT_ID"),
		Prompt: func(m auth.DeviceCodeMessage) error {
			fmt.Println(m.Message)
			return nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	sp, err := sharepoint.New(os.Getenv("SITE_URL"), cred, sharepoint.Options{})
	if err != nil {
		log.Fatal(err)
	}
	ls, err := sp.Lists().All(ctx, httpx.ODataQuery{Filter: "Hidden eq false"})
	if err != nil {
		log.Fatal(err)
	}
	for _, l := range ls {
		fmt.Println(l.Title, l.ItemCount)
	}
}
