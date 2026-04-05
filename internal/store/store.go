package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Projects struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Category string `json:"category"`
	Client string `json:"client"`
	Date string `json:"date"`
	Description string `json:"description"`
	ImageUrl string `json:"image_url"`
	ProjectUrl string `json:"project_url"`
	Tags string `json:"tags"`
	Featured bool `json:"featured"`
	Status string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil { return nil, err }
	db, err := sql.Open("sqlite", filepath.Join(d, "portfolio.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil { return nil, err }
	db.SetMaxOpenConns(1)
	db.Exec(`CREATE TABLE IF NOT EXISTS projects(id TEXT PRIMARY KEY, title TEXT NOT NULL, category TEXT DEFAULT '', client TEXT DEFAULT '', date TEXT DEFAULT '', description TEXT DEFAULT '', image_url TEXT DEFAULT '', project_url TEXT DEFAULT '', tags TEXT DEFAULT '', featured INTEGER DEFAULT 0, status TEXT DEFAULT '', created_at TEXT DEFAULT(datetime('now')))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) CreateProjects(e *Projects) error {
	e.ID = genID(); e.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO projects(id, title, category, client, date, description, image_url, project_url, tags, featured, status, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, e.Title, e.Category, e.Client, e.Date, e.Description, e.ImageUrl, e.ProjectUrl, e.Tags, e.Featured, e.Status, e.CreatedAt)
	return err
}

func (d *DB) GetProjects(id string) *Projects {
	var e Projects
	if d.db.QueryRow(`SELECT id, title, category, client, date, description, image_url, project_url, tags, featured, status, created_at FROM projects WHERE id=?`, id).Scan(&e.ID, &e.Title, &e.Category, &e.Client, &e.Date, &e.Description, &e.ImageUrl, &e.ProjectUrl, &e.Tags, &e.Featured, &e.Status, &e.CreatedAt) != nil { return nil }
	return &e
}

func (d *DB) ListProjects() []Projects {
	rows, _ := d.db.Query(`SELECT id, title, category, client, date, description, image_url, project_url, tags, featured, status, created_at FROM projects ORDER BY created_at DESC`)
	if rows == nil { return nil }; defer rows.Close()
	var o []Projects
	for rows.Next() { var e Projects; rows.Scan(&e.ID, &e.Title, &e.Category, &e.Client, &e.Date, &e.Description, &e.ImageUrl, &e.ProjectUrl, &e.Tags, &e.Featured, &e.Status, &e.CreatedAt); o = append(o, e) }
	return o
}

func (d *DB) UpdateProjects(e *Projects) error {
	_, err := d.db.Exec(`UPDATE projects SET title=?, category=?, client=?, date=?, description=?, image_url=?, project_url=?, tags=?, featured=?, status=? WHERE id=?`, e.Title, e.Category, e.Client, e.Date, e.Description, e.ImageUrl, e.ProjectUrl, e.Tags, e.Featured, e.Status, e.ID)
	return err
}

func (d *DB) DeleteProjects(id string) error {
	_, err := d.db.Exec(`DELETE FROM projects WHERE id=?`, id)
	return err
}

func (d *DB) CountProjects() int {
	var n int; d.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&n); return n
}

func (d *DB) SearchProjects(q string, filters map[string]string) []Projects {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (title LIKE ? OR category LIKE ? OR client LIKE ? OR description LIKE ? OR image_url LIKE ? OR project_url LIKE ? OR tags LIKE ?)"
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
	}
	if v, ok := filters["status"]; ok && v != "" { where += " AND status=?"; args = append(args, v) }
	rows, _ := d.db.Query(`SELECT id, title, category, client, date, description, image_url, project_url, tags, featured, status, created_at FROM projects WHERE `+where+` ORDER BY created_at DESC`, args...)
	if rows == nil { return nil }; defer rows.Close()
	var o []Projects
	for rows.Next() { var e Projects; rows.Scan(&e.ID, &e.Title, &e.Category, &e.Client, &e.Date, &e.Description, &e.ImageUrl, &e.ProjectUrl, &e.Tags, &e.Featured, &e.Status, &e.CreatedAt); o = append(o, e) }
	return o
}
