package api

import (
	"net/http"

	"github.com/unrolled/render"
)

type boxHandler struct {
	m  *Manager
	rd *render.Render
}

func newBoxHandler(m *Manager, rd *render.Render) *boxHandler {
	return &boxHandler{
		m:  m,
		rd: rd,
	}
}

func (b *boxHandler) newBox(w http.ResponseWriter, r *http.Request) {

}
