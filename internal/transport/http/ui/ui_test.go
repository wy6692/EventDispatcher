package ui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"eventdispatcher/internal/transport/http/ui"
)

func TestUIHandlerStaticFiles(t *testing.T) {
	handler := ui.Handler()

	t.Run("redirect /ui to /ui/", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusFound {
			t.Fatalf("expected status 302, got %d", rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/ui/" {
			t.Fatalf("expected Location /ui/, got %s", loc)
		}
	})

	t.Run("serve /ui/ index.html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		body, _ := io.ReadAll(rec.Body)
		if !strings.Contains(string(body), "EventDispatcher 控制台") {
			t.Fatalf("expected title in index.html, got: %s", string(body[:200]))
		}
		if !strings.Contains(string(body), "js/app.js") {
			t.Fatalf("expected script tag for js/app.js")
		}
	})

	t.Run("serve /ui/css/base.css", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/css/base.css", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		body, _ := io.ReadAll(rec.Body)
		if !strings.Contains(string(body), "--font-sans") {
			t.Fatalf("expected css variable in base.css")
		}
	})

	t.Run("serve /ui/js/app.js", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/js/app.js", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		body, _ := io.ReadAll(rec.Body)
		if !strings.Contains(string(body), "switchTab") {
			t.Fatalf("expected switchTab in app.js")
		}
	})

	t.Run("serve vendor assets locally", func(t *testing.T) {
		for _, file := range []string{"jquery.min.js", "tailwindcss.js", "sweetalert2.all.min.js"} {
			req := httptest.NewRequest(http.MethodGet, "/ui/js/vendor/"+file, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200 for /ui/js/vendor/%s, got %d", file, rec.Code)
			}
			if rec.Body.Len() == 0 {
				t.Fatalf("expected non-empty body for /ui/js/vendor/%s", file)
			}
		}
	})
}
