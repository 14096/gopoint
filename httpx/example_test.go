package httpx_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/14096/gopoint/httpx"
)

func ExampleClient_Get() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"displayName":"Team"}`)
	}))
	defer srv.Close()

	c, err := httpx.New(httpx.Config{BaseURL: srv.URL})
	if err != nil {
		fmt.Println(err)
		return
	}
	var out struct{ DisplayName string }
	if err := c.Get(context.Background(), "/sites/root", &out); err != nil {
		fmt.Println(err)
	}
	fmt.Println(out.DisplayName)
	// Output: Team
}
