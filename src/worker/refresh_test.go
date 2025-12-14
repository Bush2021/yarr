package worker

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nkanaev/yarr/src/client"
	"github.com/nkanaev/yarr/src/storage/model"
)

func TestRefreshFeedUsesClientAndConditionalRequests(t *testing.T) {
	db := newTestDB(t)
	requests := 0
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feed" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		requests++
		if got := r.Header.Get("User-Agent"); got != "test-browser" {
			t.Errorf("User-Agent = %q; want test-browser", got)
		}
		if requests > 1 {
			if got := r.Header.Get("If-None-Match"); got != "v1" {
				t.Errorf("If-None-Match = %q; want v1", got)
			}
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", "v1")
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(feedFixture))
	}))
	defer feedServer.Close()

	feed := db.CreateFeed(model.CreateFeedParams{Title: "Test", FeedLink: feedServer.URL + "/feed"})
	lastError := "previous error"
	db.UpdateFeedState(feed.Id, model.UpdateFeedStateParams{LastError: &lastError})
	httpClient := client.NewBuilder().Middleware(
		client.UserAgent("test-browser"),
		client.ConditionalRequests(NewFeedStateStore(db)),
	).Build()
	worker := NewWorker(db, httpClient)
	worker.RefreshFeed(feed)
	worker.RefreshFeed(feed)

	if requests != 2 || db.CountItems() != 1 {
		t.Fatalf("requests = %d, items = %d; want 2 requests and 1 item", requests, db.CountItems())
	}
	state, err := db.GetFeedState(feed.Id)
	if err != nil {
		t.Fatal(err)
	}
	if state == nil || state.HTTPEtag != "v1" || state.LastError != "" {
		t.Fatalf("unexpected feed state: %#v", state)
	}
}
