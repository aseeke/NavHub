package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	_ "modernc.org/sqlite"
)

type Category struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Visible bool   `json:"visible"`
}
type Item struct {
	ID          string `json:"id"`
	CategoryID  string `json:"categoryId"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	IconColor   string `json:"iconColor"`
	Target      string `json:"target"`
}
type Settings struct {
	Title           string `json:"title"`
	Background      string `json:"background"`
	BackgroundColor string `json:"backgroundColor"`
	Gradient        bool   `json:"gradient"`
	Overlay         int    `json:"overlay"`
	ShowClock       bool   `json:"showClock"`
	ShowDate        bool   `json:"showDate"`
	CardColor       string `json:"cardColor"`
	IconColor       string `json:"iconColor"`
	Target          string `json:"target"`
	Radius          int    `json:"radius"`
	Gap             int    `json:"gap"`
	Public          bool   `json:"public"`
}
type Data struct {
	FormatVersion int        `json:"formatVersion"`
	Revision      int        `json:"revision"`
	Categories    []Category `json:"categories"`
	Items         []Item     `json:"items"`
	Settings      Settings   `json:"settings"`
}

func defaults() Data {
	return Data{FormatVersion: 1, Categories: []Category{}, Items: []Item{}, Settings: Settings{Title: "NavHub", BackgroundColor: "#f5f7fa", Gradient: true, Overlay: 0, ShowClock: true, ShowDate: true, CardColor: "#ffffff", IconColor: "#526b91", Target: "_blank", Radius: 12, Gap: 14}}
}
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	fail := func(e error) (*sql.DB, error) { db.Close(); return nil, e }
	for _, q := range []string{"PRAGMA busy_timeout=5000", "PRAGMA journal_mode=WAL", "PRAGMA foreign_keys=ON"} {
		if _, err = db.Exec(q); err != nil {
			return fail(err)
		}
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	if version > 1 {
		return fail(fmt.Errorf("数据库版本 %d 高于程序支持的版本 1", version))
	}
	if version == 0 {
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		defer tx.Rollback()
		for _, q := range []string{
			"CREATE TABLE app_state (id INTEGER PRIMARY KEY CHECK(id=1), revision INTEGER NOT NULL, data TEXT NOT NULL)",
			"CREATE TABLE admin (id INTEGER PRIMARY KEY CHECK(id=1), username TEXT NOT NULL, password BLOB NOT NULL)",
			"CREATE TABLE sessions (token TEXT PRIMARY KEY, expires INTEGER NOT NULL)",
			"PRAGMA user_version=1",
		} {
			if _, e = tx.Exec(q); e != nil {
				return fail(e)
			}
		}
		raw, _ := json.Marshal(defaults())
		if _, e = tx.Exec("INSERT INTO app_state VALUES(1,0,?)", string(raw)); e != nil {
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}
	return db, nil
}
func readData(db *sql.DB) (Data, error) {
	var d Data
	var raw string
	var rev int
	err := db.QueryRow("SELECT revision,data FROM app_state WHERE id=1").Scan(&rev, &raw)
	if err != nil {
		return d, err
	}
	err = json.Unmarshal([]byte(raw), &d)
	d.Revision = rev
	return d, err
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validURL(s string, media bool) bool {
	if media && strings.HasPrefix(s, "/uploads/") {
		return regexp.MustCompile(`^/uploads/[a-f0-9]{32}\.(png|jpg|gif|webp)$`).MatchString(s)
	}
	u, e := url.Parse(s)
	return e == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil
}
func validate(d *Data) error {
	if d.FormatVersion != 1 {
		return errors.New("不支持的 JSON 格式版本")
	}
	if len(d.Categories) > 200 || len(d.Items) > 5000 {
		return errors.New("最多支持 200 个分类和 5000 个项目")
	}
	if d.Categories == nil {
		d.Categories = []Category{}
	}
	if d.Items == nil {
		d.Items = []Item{}
	}
	cats := map[string]bool{}
	ids := map[string]bool{}
	for i := range d.Categories {
		c := &d.Categories[i]
		c.Name = strings.TrimSpace(c.Name)
		if c.ID == "" || len(c.ID) > 80 || cats[c.ID] || c.Name == "" || len([]rune(c.Name)) > 60 {
			return errors.New("分类名称或 ID 无效 / 重复")
		}
		cats[c.ID] = true
	}
	for i := range d.Items {
		v := &d.Items[i]
		v.Name = strings.TrimSpace(v.Name)
		v.URL = strings.TrimSpace(v.URL)
		if v.ID == "" || len(v.ID) > 80 || ids[v.ID] || !cats[v.CategoryID] || v.Name == "" || len([]rune(v.Name)) > 100 || len(v.Description) > 2000 || len(v.URL) > 2048 || !validURL(v.URL, false) {
			return errors.New("项目名称、分类或网址无效（仅支持 HTTP / HTTPS）")
		}
		ids[v.ID] = true
		if v.Icon != "" && !validURL(v.Icon, true) {
			return errors.New("图标地址无效")
		}
		if (v.Color != "" && !hexColor.MatchString(v.Color)) || (v.IconColor != "" && !hexColor.MatchString(v.IconColor)) || (v.Target != "" && v.Target != "_blank" && v.Target != "_self") {
			return errors.New("卡片颜色或打开方式无效")
		}
	}
	s := d.Settings
	s.Title = strings.TrimSpace(s.Title)
	d.Settings.Title = s.Title
	if s.Title == "" || len([]rune(s.Title)) > 60 || !hexColor.MatchString(s.BackgroundColor) || !hexColor.MatchString(s.CardColor) || !hexColor.MatchString(s.IconColor) || s.Overlay < 0 || s.Overlay > 80 || s.Radius < 0 || s.Radius > 30 || s.Gap < 6 || s.Gap > 30 || (s.Target != "_blank" && s.Target != "_self") || (s.Background != "" && !validURL(s.Background, true)) {
		return errors.New("页面设置无效，请检查颜色、图片地址和数值范围")
	}
	return nil
}
