// Package subscriptions implements Graph change notification subscriptions
// (webhooks) and an http.Handler for receiving the notifications.
package subscriptions

import (
	"context"
	"encoding/json"
	"io"
	"iter"
	"net/http"
	"net/url"
	"time"

	"github.com/14096/gopoint/httpx"
)

// Service exposes subscription operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Subscription is a Graph change notification subscription.
type Subscription struct {
	ID                 string `json:"id,omitempty"`
	Resource           string `json:"resource"`
	ChangeType         string `json:"changeType"`
	NotificationURL    string `json:"notificationUrl"`
	ExpirationDateTime string `json:"expirationDateTime"`
	ClientState        string `json:"clientState,omitempty"`
}

// Expiry formats t as Graph expects (UTC, RFC 3339).
func Expiry(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// Create registers a subscription. Graph calls NotificationURL with a
// validation request before the call returns; see Handler.
func (s *Service) Create(ctx context.Context, sub Subscription) (*Subscription, error) {
	var out Subscription
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("POST", "/subscriptions").WithJSON(sub), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns one subscription.
func (s *Service) Get(ctx context.Context, id string) (*Subscription, error) {
	var out Subscription
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", "/subscriptions/"+url.PathEscape(id)), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Iter iterates over the app's subscriptions.
func (s *Service) Iter(ctx context.Context) iter.Seq2[Subscription, error] {
	return httpx.Paginate[Subscription](ctx, s.c, httpx.NewRequest("GET", "/subscriptions"))
}

// Renew sets a new expiration time.
func (s *Service) Renew(ctx context.Context, id string, expires time.Time) (*Subscription, error) {
	var out Subscription
	r := httpx.NewRequest("PATCH", "/subscriptions/"+url.PathEscape(id)).WithJSON(map[string]string{"expirationDateTime": Expiry(expires)})
	if err := httpx.JSON(ctx, s.c, r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a subscription.
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", "/subscriptions/"+url.PathEscape(id)))
	return err
}

// Notification is one change notification.
type Notification struct {
	SubscriptionID     string `json:"subscriptionId"`
	ClientState        string `json:"clientState"`
	ChangeType         string `json:"changeType"`
	Resource           string `json:"resource"`
	TenantID           string `json:"tenantId"`
	ExpirationDateTime string `json:"subscriptionExpirationDateTime"`
}

// Handler returns an http.Handler for the notification URL. It answers the
// validation handshake and acknowledges notifications with 202 before calling
// fn, so fn should return quickly (Graph expects a response within 3 s). When
// clientState is not empty, notifications with a different state are dropped.
func Handler(clientState string, fn func(Notification)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tok := r.URL.Query().Get("validationToken"); tok != "" {
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
		w.WriteHeader(http.StatusAccepted)
		for _, n := range body.Value {
			if clientState == "" || n.ClientState == clientState {
				fn(n)
			}
		}
	})
}
