package main

import (
	"context"
	"net/http"
	"time"
)

func registerProductRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/products", handleProducts)
	registerMLRoutes(mux)
}

// handleProducts always answers 200 with {connected, products} — "not
// connected to Mercado Livre yet" is an expected state for this MVP (see
// ml_oauth.go), not an error the frontend needs to treat as one.
func handleProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, errQueryRequired)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	products, err := searchMercadoLivre(ctx, q, 24)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if products == nil {
		writeJSON(w, http.StatusOK, map[string]any{"connected": false, "products": []Product{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"connected": true, "products": products})
}
