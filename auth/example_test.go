package auth_test

import (
	"fmt"

	"github.com/14096/gopoint/auth"
)

func ExampleNewDeviceCode() {
	cred, err := auth.NewDeviceCode(auth.DeviceCodeConfig{
		TenantID: "contoso.onmicrosoft.com",
		ClientID: "00000000-0000-0000-0000-000000000000",
		Prompt: func(m auth.DeviceCodeMessage) error {
			fmt.Println(m.Message)
			return nil
		},
	})
	_, _ = cred, err
}

func ExampleNewInteractiveBrowser() {
	cred, err := auth.NewInteractiveBrowser(auth.BrowserConfig{
		TenantID: "contoso.onmicrosoft.com",
		ClientID: "00000000-0000-0000-0000-000000000000",
	})
	_, _ = cred, err
}

func ExampleNewCertificateFromFile() {
	cred, err := auth.NewCertificateFromFile(
		"contoso.onmicrosoft.com", "00000000-0000-0000-0000-000000000000",
		"app.pfx", "pfx-password", auth.Options{})
	_, _ = cred, err
}
