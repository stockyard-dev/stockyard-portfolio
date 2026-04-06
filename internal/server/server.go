package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/stockyard-dev/stockyard-portfolio/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}
	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/projects", s.listProjects)
	s.mux.HandleFunc("POST /api/projects", s.createProjects)
	s.mux.HandleFunc("GET /api/projects/export.csv", s.exportProjects)
	s.mux.HandleFunc("GET /api/projects/{id}", s.getProjects)
	s.mux.HandleFunc("PUT /api/projects/{id}", s.updateProjects)
	s.mux.HandleFunc("DELETE /api/projects/{id}", s.delProjects)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", s.tierHandler)
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func wj(w http.ResponseWriter, c int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(c); json.NewEncoder(w).Encode(v) }
func we(w http.ResponseWriter, c int, m string) { wj(w, c, map[string]string{"error": m}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) { if r.URL.Path != "/" { http.NotFound(w, r); return }; http.Redirect(w, r, "/ui", 302) }
func oe[T any](s []T) []T { if s == nil { return []T{} }; return s }
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filters := map[string]string{}
	if v := r.URL.Query().Get("status"); v != "" { filters["status"] = v }
	if q != "" || len(filters) > 0 { wj(w, 200, map[string]any{"projects": oe(s.db.SearchProjects(q, filters))}); return }
	wj(w, 200, map[string]any{"projects": oe(s.db.ListProjects())})
}

func (s *Server) createProjects(w http.ResponseWriter, r *http.Request) {
	if s.limits.Tier == "none" { we(w, 402, "No license key. Start a 14-day trial at https://stockyard.dev/for/"); return }
	if s.limits.TrialExpired { we(w, 402, "Trial expired. Subscribe at https://stockyard.dev/pricing/"); return }
	var e store.Projects
	json.NewDecoder(r.Body).Decode(&e)
	if e.Title == "" { we(w, 400, "title required"); return }
	s.db.CreateProjects(&e)
	wj(w, 201, s.db.GetProjects(e.ID))
}

func (s *Server) getProjects(w http.ResponseWriter, r *http.Request) {
	e := s.db.GetProjects(r.PathValue("id"))
	if e == nil { we(w, 404, "not found"); return }
	wj(w, 200, e)
}

func (s *Server) updateProjects(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetProjects(r.PathValue("id"))
	if existing == nil { we(w, 404, "not found"); return }
	var patch store.Projects
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID; patch.CreatedAt = existing.CreatedAt
	if patch.Title == "" { patch.Title = existing.Title }
	if patch.Category == "" { patch.Category = existing.Category }
	if patch.Client == "" { patch.Client = existing.Client }
	if patch.Date == "" { patch.Date = existing.Date }
	if patch.Description == "" { patch.Description = existing.Description }
	if patch.ImageUrl == "" { patch.ImageUrl = existing.ImageUrl }
	if patch.ProjectUrl == "" { patch.ProjectUrl = existing.ProjectUrl }
	if patch.Tags == "" { patch.Tags = existing.Tags }
	if patch.Status == "" { patch.Status = existing.Status }
	s.db.UpdateProjects(&patch)
	wj(w, 200, s.db.GetProjects(patch.ID))
}

func (s *Server) delProjects(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteProjects(r.PathValue("id"))
	wj(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) exportProjects(w http.ResponseWriter, r *http.Request) {
	items := s.db.ListProjects()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=projects.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"id", "title", "category", "client", "date", "description", "image_url", "project_url", "tags", "featured", "status", "created_at"})
	for _, e := range items {
		cw.Write([]string{e.ID, fmt.Sprintf("%v", e.Title), fmt.Sprintf("%v", e.Category), fmt.Sprintf("%v", e.Client), fmt.Sprintf("%v", e.Date), fmt.Sprintf("%v", e.Description), fmt.Sprintf("%v", e.ImageUrl), fmt.Sprintf("%v", e.ProjectUrl), fmt.Sprintf("%v", e.Tags), fmt.Sprintf("%v", e.Featured), fmt.Sprintf("%v", e.Status), e.CreatedAt})
	}
	cw.Flush()
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{}
	m["projects_total"] = s.db.CountProjects()
	wj(w, 200, m)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{"status": "ok", "service": "portfolio"}
	m["projects"] = s.db.CountProjects()
	wj(w, 200, m)
}

// loadPersonalConfig reads config.json from the data directory.
func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("warning: could not parse config.json: %v", err)
		return
	}
	s.pCfg = cfg
	log.Printf("loaded personalization from %s", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		wj(w, 200, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}
