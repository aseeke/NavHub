package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNavigationLifecycle(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "uploads"), 0700)
	os.MkdirAll(filepath.Join(dir, "backups"), 0700)
	db, e := openDB(filepath.Join(dir, "navhub.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer func() { db.Close() }()
	a := &app{db: db, dir: dir, attempts: map[string]attempt{}}
	h := a.handler()
	cookie := ""
	request := func(method, path string, body any, authenticated bool) *httptest.ResponseRecorder {
		t.Helper()
		var b io.Reader
		if body != nil {
			raw, _ := json.Marshal(body)
			b = bytes.NewReader(raw)
		}
		r := httptest.NewRequest(method, path, b)
		if authenticated && cookie != "" {
			r.Header.Set("Cookie", cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	expect := func(w *httptest.ResponseRecorder, status int) {
		t.Helper()
		if w.Code != status {
			t.Fatalf("want %d got %d: %s", status, w.Code, w.Body.String())
		}
	}
	expect(request("GET", "/api/data", nil, false), 401)
	expect(request("POST", "/api/setup", map[string]string{"username": "admin", "password": "short"}, false), 400)
	w := request("POST", "/api/setup", map[string]string{"username": "admin", "password": "test-password-12345"}, false)
	expect(w, 200)
	cookie = w.Result().Cookies()[0].String()
	expect(request("POST", "/api/setup", map[string]string{"username": "other", "password": "test-password-12345"}, false), 409)
	expect(request("POST", "/api/login", map[string]string{"username": "admin", "password": "wrong"}, false), 401)
	expect(request("PUT", "/api/data", defaults(), false), 401)
	d := defaults()
	d.Categories = []Category{{"work", "工作", true}, {"cloud", "云端", true}, {"hidden", "隐藏", false}}
	d.Items = []Item{{ID: "github", CategoryID: "work", Name: "GitHub", URL: "https://github.com", Color: "#ffffff"}, {ID: "cloud", CategoryID: "cloud", Name: "Oracle Cloud", URL: "https://cloud.oracle.com"}, {ID: "secret", CategoryID: "hidden", Name: "私有系统", URL: "https://private.example.com"}}
	w = request("PUT", "/api/data", d, true)
	expect(w, 200)
	json.Unmarshal(w.Body.Bytes(), &d)
	stale := d
	stale.Revision = 0
	expect(request("PUT", "/api/data", stale, true), 409)
	invalid := d
	invalid.Items = append([]Item{}, d.Items...)
	invalid.Items[0].URL = "javascript:alert(1)"
	expect(request("PUT", "/api/data", invalid, true), 400)
	invalid = d
	invalid.Categories = invalid.Categories[1:]
	invalid.Items = invalid.Items[1:]
	expect(request("PUT", "/api/data", invalid, true), 400)
	// Reorder and move across categories, persisted as an atomic snapshot.
	d.Items[0], d.Items[1] = d.Items[1], d.Items[0]
	d.Items[0].CategoryID = "work"
	d.Categories[0], d.Categories[1] = d.Categories[1], d.Categories[0]
	d.Categories[0].Name = "云端服务"
	w = request("PUT", "/api/data", d, true)
	expect(w, 200)
	json.Unmarshal(w.Body.Bytes(), &d)
	// Upload a real image and persist its reference.
	imageData := new(bytes.Buffer)
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.White)
	png.Encode(imageData, img)
	body := new(bytes.Buffer)
	form := multipart.NewWriter(body)
	part, _ := form.CreateFormFile("file", "test.png")
	part.Write(imageData.Bytes())
	form.Close()
	r := httptest.NewRequest("POST", "/api/upload", body)
	r.Header.Set("Cookie", cookie)
	r.Header.Set("Content-Type", form.FormDataContentType())
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	expect(w, 200)
	var uploaded map[string]string
	json.Unmarshal(w.Body.Bytes(), &uploaded)
	d.Items[0].Icon = uploaded["url"]
	w = request("PUT", "/api/data", d, true)
	expect(w, 200)
	json.Unmarshal(w.Body.Bytes(), &d)
	expect(request("GET", uploaded["url"], nil, false), 401)
	// Restart the database; sessions, media, order and navigation survive.
	db.Close()
	db, e = openDB(filepath.Join(dir, "navhub.db"))
	if e != nil {
		t.Fatal(e)
	}
	a.db = db
	w = request("GET", "/api/data", nil, true)
	expect(w, 200)
	var restarted Data
	json.Unmarshal(w.Body.Bytes(), &restarted)
	if restarted.Items[0].ID != "cloud" || restarted.Items[0].CategoryID != "work" {
		t.Fatal("order/move lost")
	}
	if restarted.Categories[0].ID != "cloud" || restarted.Categories[0].Name != "云端服务" {
		t.Fatal("category order/edit lost")
	}
	w = request("GET", uploaded["url"], nil, true)
	expect(w, 200)
	if !bytes.Equal(w.Body.Bytes(), imageData.Bytes()) {
		t.Fatal("media lost")
	}
	d.Settings.Public = true
	w = request("PUT", "/api/data", d, true)
	expect(w, 200)
	json.Unmarshal(w.Body.Bytes(), &d)
	w = request("GET", "/api/data", nil, false)
	expect(w, 200)
	var public Data
	json.Unmarshal(w.Body.Bytes(), &public)
	if len(public.Items) != 2 || len(public.Categories) != 2 {
		t.Fatal("hidden data leaked")
	}
	// Public access is limited to referenced visible media.
	expect(request("GET", uploaded["url"], nil, false), 200)
	os.WriteFile(filepath.Join(dir, "uploads", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png"), imageData.Bytes(), 0600)
	expect(request("GET", "/uploads/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png", nil, false), 404)
	// JSON round trip contains no credential/session tables.
	w = request("GET", "/api/export", nil, true)
	expect(w, 200)
	if bytes.Contains(w.Body.Bytes(), []byte("test-password")) || bytes.Contains(w.Body.Bytes(), []byte("navhub_session")) {
		t.Fatal("credentials leaked")
	}
	var exported Data
	json.Unmarshal(w.Body.Bytes(), &exported)
	w = request("POST", "/api/import", exported, true)
	expect(w, 200)
	// Full ZIP restores a consistent database and original uploaded media.
	w = request("POST", "/api/backup", nil, true)
	expect(w, 200)
	z, e := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if e != nil {
		t.Fatal(e)
	}
	restoredDir := t.TempDir()
	for _, f := range z.File {
		src, _ := f.Open()
		b, _ := io.ReadAll(src)
		src.Close()
		p := filepath.Join(restoredDir, filepath.FromSlash(f.Name))
		os.MkdirAll(filepath.Dir(p), 0700)
		os.WriteFile(p, b, 0600)
	}
	restored, e := openDB(filepath.Join(restoredDir, "navhub.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	restoredData, e := readData(restored)
	if e != nil || len(restoredData.Items) != 3 {
		t.Fatal("backup restore failed", e)
	}
	if _, e = os.Stat(filepath.Join(restoredDir, filepath.FromSlash(uploaded["url"][1:]))); e != nil {
		t.Fatal(e)
	}
	expect(request("GET", "/settings/profile", nil, false), 200)
	expect(request("GET", "/api/missing", nil, false), 404)
	r = httptest.NewRequest("POST", "/api/logout", nil)
	r.Header.Set("Cookie", cookie)
	r.Header.Set("Origin", "https://evil.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	expect(w, 403)
	expect(request("POST", "/api/logout", nil, true), 200)
	expect(request("PUT", "/api/data", d, true), 401)
}

func TestMigrationsPreserveDataAndRejectFutureSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "navhub.db")
	db, e := openDB(path)
	if e != nil {
		t.Fatal(e)
	}
	d := defaults()
	d.Settings.Title = "保留的数据"
	raw, _ := json.Marshal(d)
	db.Exec("UPDATE app_state SET data=?", string(raw))
	db.Close()
	db, e = openDB(path)
	if e != nil {
		t.Fatal(e)
	}
	got, e := readData(db)
	if e != nil || got.Settings.Title != d.Settings.Title {
		t.Fatal("migration lost data")
	}
	db.Exec("PRAGMA user_version=99")
	db.Close()
	if future, e := openDB(path); e == nil {
		future.Close()
		t.Fatal("accepted unsupported database")
	}
	check, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	defer check.Close()
	var count int
	if e = check.QueryRow("SELECT count(*) FROM app_state").Scan(&count); e != nil || count != 1 {
		t.Fatal("database modified after migration failure", e)
	}
}

func TestLoginRateLimit(t *testing.T) {
	db, e := openDB(filepath.Join(t.TempDir(), "navhub.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	a := &app{db: db, attempts: map[string]attempt{}}
	h := a.handler()
	for i := 0; i < 11; i++ {
		r := httptest.NewRequest("POST", "/api/login", bytes.NewBufferString(`{"username":"x","password":"wrong"}`))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if i == 10 && w.Code != http.StatusTooManyRequests {
			t.Fatal("missing rate limit")
		}
	}
}

func TestLogRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "navhub.log")
	f, e := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	l := &rotatingLog{path: path, file: f}
	defer func() { l.file.Close() }()
	for range 5 {
		l.file.Close()
		if e = os.Truncate(path, (5<<20)-3); e != nil {
			t.Fatal(e)
		}
		l.file, e = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = l.Write([]byte("hello")); e != nil {
			t.Fatal(e)
		}
	}
	if b, e := os.ReadFile(path); e != nil || string(b) != "hello" {
		t.Fatal("current log invalid", e)
	}
	if _, e = os.Stat(path + ".3"); e != nil {
		t.Fatal("missing retained log", e)
	}
	if _, e = os.Stat(path + ".4"); !os.IsNotExist(e) {
		t.Fatal("too many retained logs")
	}
}
