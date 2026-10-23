package main

import (
	"archive/zip"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//go:embed static
var assets embed.FS

type attempt struct {
	count int
	until time.Time
}
type app struct {
	db       *sql.DB
	dir      string
	secure   bool
	mu       sync.Mutex
	attempts map[string]attempt
}

func randomID() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func tokenHash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, msg string) {
	reply(w, status, map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return errors.New("JSON 内容无效或超过 4MB")
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return errors.New("JSON 包含多余内容")
	}
	return nil
}
func (a *app) authenticated(r *http.Request) bool {
	c, e := r.Cookie("navhub_session")
	if e != nil || len(c.Value) != 64 {
		return false
	}
	var expires int64
	e = a.db.QueryRow("SELECT expires FROM sessions WHERE token=?", tokenHash(c.Value)).Scan(&expires)
	return e == nil && expires > time.Now().Unix()
}
func (a *app) initialized() bool {
	var n int
	a.db.QueryRow("SELECT count(*) FROM admin").Scan(&n)
	return n > 0
}
func (a *app) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		d, e := readData(a.db)
		if e != nil {
			a.failure(w, e)
			return
		}
		reply(w, 200, map[string]any{"initialized": a.initialized(), "authenticated": a.authenticated(r), "public": d.Settings.Public, "title": d.Settings.Title, "version": version})
	})
	mux.HandleFunc("POST /api/setup", a.login)
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		if c, e := r.Cookie("navhub_session"); e == nil {
			a.db.Exec("DELETE FROM sessions WHERE token=?", tokenHash(c.Value))
		}
		http.SetCookie(w, &http.Cookie{Name: "navhub_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.secure || r.TLS != nil, SameSite: http.SameSiteStrictMode})
		reply(w, 200, map[string]bool{"ok": true})
	})
	mux.HandleFunc("GET /api/data", func(w http.ResponseWriter, r *http.Request) {
		d, e := readData(a.db)
		if e != nil {
			a.failure(w, e)
			return
		}
		admin := a.authenticated(r)
		if !admin && !d.Settings.Public {
			problem(w, 401, "请先登录")
			return
		}
		if !admin {
			visible := map[string]bool{}
			cats := []Category{}
			items := []Item{}
			for _, c := range d.Categories {
				if c.Visible {
					visible[c.ID] = true
					cats = append(cats, c)
				}
			}
			for _, i := range d.Items {
				if visible[i.CategoryID] {
					items = append(items, i)
				}
			}
			d.Categories = cats
			d.Items = items
		}
		reply(w, 200, d)
	})
	mux.HandleFunc("PUT /api/data", a.admin(a.save))
	mux.HandleFunc("POST /api/import", a.admin(a.save))
	mux.HandleFunc("GET /api/export", a.admin(func(w http.ResponseWriter, r *http.Request) {
		d, e := readData(a.db)
		if e != nil {
			a.failure(w, e)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="navhub.json"`)
		reply(w, 200, d)
	}))
	mux.HandleFunc("POST /api/upload", a.admin(a.upload))
	mux.HandleFunc("POST /api/backup", a.admin(a.backup))
	mux.HandleFunc("GET /uploads/", func(w http.ResponseWriter, r *http.Request) {
		d, e := readData(a.db)
		if e != nil {
			a.failure(w, e)
			return
		}
		if !a.authenticated(r) {
			if !d.Settings.Public {
				problem(w, 401, "请先登录")
				return
			}
			allowed := d.Settings.Background == r.URL.Path
			visible := map[string]bool{}
			for _, c := range d.Categories {
				visible[c.ID] = c.Visible
			}
			for _, i := range d.Items {
				if visible[i.CategoryID] && i.Icon == r.URL.Path {
					allowed = true
				}
			}
			if !allowed {
				http.NotFound(w, r)
				return
			}
		}
		name := strings.TrimPrefix(r.URL.Path, "/uploads/")
		if !validURL("/uploads/"+name, true) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.ServeFile(w, r, filepath.Join(a.dir, "uploads", name))
	})
	sub, _ := fs.Sub(assets, "static")
	files := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/uploads/") {
			http.NotFound(w, r)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		if _, e := fs.Stat(sub, p); e != nil {
			if filepath.Ext(p) != "" {
				http.NotFound(w, r)
				return
			}
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				log.Printf("系统异常: %v", e)
				problem(w, 500, "系统异常")
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https: http: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			origin := r.Header.Get("Origin")
			scheme := "http"
			if r.TLS != nil || a.secure {
				scheme = "https"
			}
			if (origin != "" && origin != scheme+"://"+r.Host) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				problem(w, 403, "请求来源不允许")
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func (a *app) failure(w http.ResponseWriter, e error) {
	log.Printf("管理操作 / 系统异常: %v", e)
	problem(w, 500, "操作失败，请查看服务端日志")
}
func (a *app) admin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authenticated(r) {
			problem(w, 401, "请先登录管理员账户")
			return
		}
		h(w, r)
	}
}
func (a *app) login(w http.ResponseWriter, r *http.Request) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	a.mu.Lock()
	now := time.Now()
	for k, v := range a.attempts {
		if now.After(v.until) {
			delete(a.attempts, k)
		}
	}
	v := a.attempts[ip]
	if v.count >= 10 {
		a.mu.Unlock()
		problem(w, 429, "尝试过多，请在 10 分钟后重试")
		return
	}
	v.count++
	if v.until.IsZero() {
		v.until = now.Add(10 * time.Minute)
	}
	a.attempts[ip] = v
	a.mu.Unlock()
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if e := decode(w, r, &in); e != nil {
		problem(w, 400, e.Error())
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if r.URL.Path == "/api/setup" {
		if in.Username == "" || len(in.Username) > 80 || len(in.Password) < 10 || len(in.Password) > 72 {
			problem(w, 400, "账户必填，密码需为 10–72 字节")
			return
		}
		hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if e != nil {
			a.failure(w, e)
			return
		}
		res, e := a.db.Exec("INSERT OR IGNORE INTO admin(id,username,password) VALUES(1,?,?)", in.Username, hash)
		if e != nil {
			a.failure(w, e)
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			problem(w, 409, "已完成初始化，请登录")
			return
		}
		log.Print("管理员初始化完成")
	} else {
		var name string
		var hash []byte
		e := a.db.QueryRow("SELECT username,password FROM admin WHERE id=1").Scan(&name, &hash)
		// Always perform a password check, even for an unknown account.
		if e != nil {
			hash = []byte("$2a$10$7EqJtq98hPqEX7fNZaFWoO5TtPpNj5pD7vXwb61fyVeIYNc3BYA6i")
		}
		match := bcrypt.CompareHashAndPassword(hash, []byte(in.Password)) == nil
		if e != nil || !match || name != in.Username {
			log.Print("登录失败")
			problem(w, 401, "账户或密码不正确")
			return
		}
	}
	token := randomID()
	expiry := time.Now().Add(7 * 24 * time.Hour)
	a.db.Exec("DELETE FROM sessions WHERE expires<?", time.Now().Unix())
	if _, e := a.db.Exec("INSERT INTO sessions VALUES(?,?)", tokenHash(token), expiry.Unix()); e != nil {
		a.failure(w, e)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "navhub_session", Value: token, Path: "/", Expires: expiry, MaxAge: 604800, HttpOnly: true, Secure: a.secure || r.TLS != nil, SameSite: http.SameSiteStrictMode})
	reply(w, 200, map[string]bool{"ok": true})
}
func (a *app) save(w http.ResponseWriter, r *http.Request) {
	var d Data
	if e := decode(w, r, &d); e != nil {
		problem(w, 400, e.Error())
		return
	}
	if e := validate(&d); e != nil {
		log.Print("管理操作失败: 数据校验失败")
		problem(w, 400, e.Error())
		return
	}
	tx, e := a.db.Begin()
	if e != nil {
		a.failure(w, e)
		return
	}
	defer tx.Rollback()
	// A category cannot be removed while any of its original items still refer to it.
	if r.URL.Path != "/api/import" {
		var raw string
		if e = tx.QueryRow("SELECT data FROM app_state WHERE id=1").Scan(&raw); e != nil {
			a.failure(w, e)
			return
		}
		var old Data
		if e = json.Unmarshal([]byte(raw), &old); e != nil {
			a.failure(w, e)
			return
		}
		cats := map[string]bool{}
		moved := map[string]bool{}
		for _, c := range d.Categories {
			cats[c.ID] = true
		}
		for _, i := range d.Items {
			moved[i.ID] = cats[i.CategoryID]
		}
		for _, i := range old.Items {
			if !cats[i.CategoryID] && !moved[i.ID] {
				problem(w, 400, "请先迁移或单独删除分类内的项目，再删除分类")
				return
			}
		}
	}
	raw, e := json.Marshal(d)
	if e != nil {
		a.failure(w, e)
		return
	}
	res, e := tx.Exec("UPDATE app_state SET data=?,revision=revision+1 WHERE id=1 AND revision=?", string(raw), d.Revision)
	if e != nil {
		a.failure(w, e)
		return
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		problem(w, 409, "数据已在其他窗口更新，请重新加载后再保存")
		return
	}
	if e = tx.Commit(); e != nil {
		a.failure(w, e)
		return
	}
	d.Revision++
	log.Printf("管理操作完成: %s", r.URL.Path)
	reply(w, 200, d)
}
func (a *app) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if e := r.ParseMultipartForm(6 << 20); e != nil {
		problem(w, 400, "图片不能超过 5MB")
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, h, e := r.FormFile("file")
	if e != nil {
		problem(w, 400, "请选择图片")
		return
	}
	defer f.Close()
	if h.Size > 5<<20 {
		problem(w, 400, "图片不能超过 5MB")
		return
	}
	b, e := io.ReadAll(io.LimitReader(f, (5<<20)+1))
	if e != nil {
		a.failure(w, e)
		return
	}
	ext := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}[http.DetectContentType(b)]
	if ext == "" {
		problem(w, 400, "支持 PNG、JPEG、GIF 和 WebP 图片")
		return
	}
	name := randomID()[:32] + ext
	if e = os.WriteFile(filepath.Join(a.dir, "uploads", name), b, 0600); e != nil {
		a.failure(w, e)
		return
	}
	reply(w, 200, map[string]string{"url": "/uploads/" + name})
}
func (a *app) backup(w http.ResponseWriter, r *http.Request) {
	name := "navhub-" + time.Now().Format("20060102-150405") + "-" + randomID()[:8]
	snapshot := filepath.Join(a.dir, "backups", name+".db")
	defer os.Remove(snapshot)
	if _, e := a.db.Exec("VACUUM INTO ?", snapshot); e != nil {
		a.failure(w, e)
		return
	}
	path := filepath.Join(a.dir, "backups", name+".zip")
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		a.failure(w, e)
		return
	}
	z := zip.NewWriter(f)
	add := func(path, name string) error {
		src, e := os.Open(path)
		if e != nil {
			return e
		}
		defer src.Close()
		dst, e := z.Create(name)
		if e != nil {
			return e
		}
		_, e = io.Copy(dst, src)
		return e
	}
	e = add(snapshot, "navhub.db")
	if e == nil {
		e = filepath.WalkDir(filepath.Join(a.dir, "uploads"), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			return add(p, "uploads/"+d.Name())
		})
	}
	e = errors.Join(e, z.Close(), f.Close())
	if e != nil {
		os.Remove(path)
		a.failure(w, e)
		return
	}
	log.Print("完整备份已创建")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, name))
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, path)
}
