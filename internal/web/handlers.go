package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

// ── auth ────────────────────────────────────────────────────────────────────

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	if !s.auth.checkCredentials(req.User, req.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	s.auth.setCookie(w, s.secure)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": req.User, "currency": s.currency})
}

func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	clearCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(sessionCookie)
	authed := err == nil && s.auth.verify(c.Value)
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": authed, "currency": s.currency})
}

// ── stats / listings ────────────────────────────────────────────────────────

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.Stats(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListItems(r.Context(), false)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListKits(w http.ResponseWriter, r *http.Request) {
	kits, err := s.store.ListKits(r.Context(), false)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, kits)
}

func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accts, err := s.store.ListLinkedAccounts(r.Context(), 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, accts)
}

func (s *Server) handleRecentTransactions(w http.ResponseWriter, r *http.Request) {
	txns, err := s.store.RecentTransactions(r.Context(), 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, txns)
}

// ── catalogue mutations ─────────────────────────────────────────────────────

func (s *Server) handleUpsertItem(w http.ResponseWriter, r *http.Request) {
	var it store.CatalogItem
	if err := decode(r, &it); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	if it.Name == "" || it.GameItemID == "" || it.Price < 0 {
		writeErr(w, http.StatusBadRequest, "name, game_item_id and non-negative price required")
		return
	}
	if it.Quantity < 1 {
		it.Quantity = 1
	}
	id, err := s.store.UpsertItem(r.Context(), &it)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *Server) handleSetItemEnabled(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	if err := s.store.SetItemEnabled(r.Context(), id, req.Enabled); errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "item not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ── kit mutations ───────────────────────────────────────────────────────────

func (s *Server) handleCreateKit(w http.ResponseWriter, r *http.Request) {
	var k store.Kit
	if err := decode(r, &k); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	if k.Name == "" || k.Price < 0 {
		writeErr(w, http.StatusBadRequest, "name and non-negative price required")
		return
	}
	id, err := s.store.CreateKit(r.Context(), &k)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *Server) handleAddKitItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var it store.KitItem
	if err := decode(r, &it); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	if it.GameItemID == "" {
		writeErr(w, http.StatusBadRequest, "game_item_id required")
		return
	}
	if it.Quantity < 1 {
		it.Quantity = 1
	}
	if err := s.store.AddKitItem(r.Context(), id, it); errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "kit not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleSetKitEnabled(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	if err := s.store.SetKitEnabled(r.Context(), id, req.Enabled); errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "kit not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}
