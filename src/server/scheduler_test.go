package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/nkanaev/yarr/src/storage"
	"github.com/nkanaev/yarr/src/storage/model"
)

type fakeScheduler struct {
	pending         int32
	refreshes       int
	rates           []int64
	singleRefreshes []int64
}

func (f *fakeScheduler) FeedsPending() int32 { return f.pending }
func (f *fakeScheduler) RefreshFeeds()       { f.refreshes++ }
func (f *fakeScheduler) RefreshFeed(feed *model.Feed) {
	f.singleRefreshes = append(f.singleRefreshes, feed.Id)
}
func (f *fakeScheduler) SetRefreshRate(min int64) {
	f.rates = append(f.rates, min)
}

func newTestServer(t *testing.T, sched FeedScheduler) http.Handler {
	t.Helper()
	db, err := storage.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer("127.0.0.1:8000")
	server.Storage = NewLocalStorage(db)
	server.Scheduler = sched
	return server.Handler()
}

func TestSchedulerStatus(t *testing.T) {
	sched := &fakeScheduler{pending: 3}
	handler := newTestServer(t, sched)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/status", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Result().StatusCode)
	}
	var body map[string]any
	json.NewDecoder(recorder.Result().Body).Decode(&body)
	if body["refresh"] != true {
		t.Errorf("expected refresh true, got %v", body["refresh"])
	}
	if running := body["running"].(float64); running != 3 {
		t.Errorf("expected running 3, got %v", running)
	}
}

func TestSchedulerRefreshFeeds(t *testing.T) {
	sched := &fakeScheduler{}
	handler := newTestServer(t, sched)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/feeds/refresh", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Result().StatusCode)
	}
	if sched.refreshes != 1 {
		t.Errorf("expected 1 refresh, got %d", sched.refreshes)
	}
}

func TestSchedulerRefreshSingleFeed(t *testing.T) {
	db, err := storage.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	feed := db.CreateFeed(model.CreateFeedParams{Title: "Test", FeedLink: "https://example.com/feed"})
	sched := &fakeScheduler{}
	server := NewServer("127.0.0.1:8000")
	server.Storage = NewLocalStorage(db)
	server.Scheduler = sched
	handler := server.Handler()

	for _, tc := range []struct {
		method string
		id     string
		status int
	}{
		{http.MethodPost, strconv.FormatInt(feed.Id, 10), http.StatusOK},
		{http.MethodPost, "invalid", http.StatusBadRequest},
		{http.MethodPost, "9999", http.StatusNotFound},
		{http.MethodGet, strconv.FormatInt(feed.Id, 10), http.StatusMethodNotAllowed},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, "/api/feeds/"+tc.id+"/refresh", nil))
		if recorder.Code != tc.status {
			t.Errorf("%s %s: status = %d; want %d", tc.method, tc.id, recorder.Code, tc.status)
		}
	}
	if len(sched.singleRefreshes) != 1 || sched.singleRefreshes[0] != feed.Id {
		t.Fatalf("single refreshes = %v; want [%d]", sched.singleRefreshes, feed.Id)
	}
}

func TestSchedulerSetRefreshRate(t *testing.T) {
	sched := &fakeScheduler{}
	handler := newTestServer(t, sched)

	body := strings.NewReader(`{"refresh_rate":15}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("PUT", "/api/settings", body)
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Result().StatusCode)
	}
	if len(sched.rates) != 1 || sched.rates[0] != 15 {
		t.Errorf("expected SetRefreshRate(15), got %v", sched.rates)
	}
}

func TestSchedulerDisabled(t *testing.T) {
	handler := newTestServer(t, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/feeds/refresh", nil)
	handler.ServeHTTP(recorder, request)
	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected refresh 200, got %d", recorder.Result().StatusCode)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest("GET", "/api/status", nil)
	handler.ServeHTTP(recorder, request)

	var body map[string]any
	json.NewDecoder(recorder.Result().Body).Decode(&body)
	if body["refresh"] != false {
		t.Errorf("expected refresh false, got %v", body["refresh"])
	}
	if running := body["running"].(float64); running != 0 {
		t.Errorf("expected running 0, got %v", running)
	}
}
