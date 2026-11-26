// Package webhooks implements SharePoint list webhooks (subscriptions) and an
// http.Handler for receiving their notifications.
package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes list webhook operations.
type Service struct {
	c       httpx.Doer
	siteURL string
}

// New returns the service; siteURL is the site the client is scoped to.
func New(c httpx.Doer, siteURL string) *Service { return &Service{c: c, siteURL: siteURL} }

// Subscription is a list webhook.
type Subscription struct {
	ID                 string `json:"id"`
	Resource           string `json:"resource"`
	NotificationURL    string `json:"notificationUrl"`
	ExpirationDateTime string `json:"expirationDateTime"`
	ClientState        string `json:"clientState"`
}

// MaxLifetime is the longest expiry SharePoint accepts (180 days).
const MaxLifetime = 180 * 24 * time.Hour

// Expiry formats t as SharePoint expects.
func Expiry(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.0000000Z") }

func subs(list spx.ListRef) string { return list.Path() + "/subscriptions" }

// List returns the webhooks of a list.
func (s *Service) List(ctx context.Context, list spx.ListRef) ([]Subscription, error) {
	return httpx.Collect[Subscription](ctx, s.c, spx.Get(subs(list)))
}

// Add registers a webhook. SharePoint calls notificationURL with a
// validationtoken query parameter before the call returns; see Handler.
func (s *Service) Add(ctx context.Context, list spx.ListRef, notificationURL string, expires time.Time, clientState string) (*Subscription, error) {
	body := map[string]string{
		"resource":           s.siteURL + "/_api/" + list.Path(),
		"notificationUrl":    notificationURL,
		"expirationDateTime": Expiry(expires),
		"clientState":        clientState,
	}
	var out Subscription
	if err := httpx.JSON(ctx, s.c, spx.Post(subs(list), body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Renew sets a new expiration time.
func (s *Service) Renew(ctx context.Context, list spx.ListRef, id string, expires time.Time) error {
	req := httpx.NewRequest(http.MethodPatch, subs(list)+"("+spx.Quote(id)+")").
		WithJSON(map[string]string{"expirationDateTime": Expiry(expires)}).
		SetHeader("Content-Type", spx.JSONContentType)
	_, err := s.c.Do(ctx, req)
	return err
}

// Delete removes a webhook.
func (s *Service) Delete(ctx context.Context, list spx.ListRef, id string) error {
	_, err := s.c.Do(ctx, spx.Remove(subs(list)+"("+spx.Quote(id)+")"))
	return err
}

// Notification is one webhook notification. It only says that something
// changed; use the list's change log (GetChanges) to find out what.
type Notification struct {
	SubscriptionID     string `json:"subscriptionId"`
	ClientState        string `json:"clientState"`
	ExpirationDateTime string `json:"expirationDateTime"`
	Resource           string `json:"resource"`
	TenantID           string `json:"tenantId"`
	SiteURL            string `json:"siteUrl"`
	WebID              string `json:"webId"`
}

// Handler returns an http.Handler for the notification URL. It echoes the
// validation token and acknowledges notifications with 200 before calling fn
// (SharePoint expects a reply within 5 s). When clientState is not empty,
// notifications with a different state are dropped.
func Handler(clientState string, fn func(Notification)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tok := r.URL.Query().Get("validationtoken"); tok != "" {
			w.Header().Set("Content-Type", "text/plain")
			io.WriteString(w, tok)
			return
		}
		var body struct {
			Value []Notification `json:"value"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		for _, n := range body.Value {
			if clientState == "" || n.ClientState == clientState {
				fn(n)
			}
		}
	})
}
