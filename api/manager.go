package api

import (
	"context"
	"net/http"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/gorilla/mux"
	"github.com/pingcap/schrodinger/cluster"
	"github.com/unrolled/render"
	"github.com/urfave/negroni"
	"k8s.io/client-go/kubernetes"
)

const APIPrefix = "/manager"

type Manager struct {
	addr       string
	s          *http.Server
	k8sCli     *kubernetes.Clientset
	etcdCli    *clientv3.Client
	clusterCli cluster.Client
}

func NewManager(addr string, etcdCli *clientv3.Client, k8sCli *kubernetes.Clientset, clusterCli cluster.Client) *Manager {
	n := &Manager{
		addr:       addr,
		etcdCli:    etcdCli,
		k8sCli:     k8sCli,
		clusterCli: clusterCli,
	}
	return n
}

func (m *Manager) Run() error {
	m.s = &http.Server{
		Addr:    m.addr,
		Handler: m.createHandler(),
	}
	return m.s.ListenAndServe()
}

func (m *Manager) Close() {
	if m.s != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		m.s.Shutdown(ctx)
		cancel()
	}
}

func (m *Manager) createHandler() http.Handler {
	engine := negroni.New()
	recover := negroni.NewRecovery()
	engine.Use(recover)

	router := mux.NewRouter()
	subRouter := m.createRouter()
	router.PathPrefix(APIPrefix).Handler(
		negroni.New(negroni.Wrap(subRouter)),
	)
	engine.UseHandler(router)
	return engine
}

func (m *Manager) createRouter() *mux.Router {
	rd := render.New(render.Options{
		IndentJSON: true,
	})
	router := mux.NewRouter().Path(APIPrefix).Subrouter()
	boxHandler := newBoxHandler(m, rd)
	router.HandleFunc("/box/new", boxHandler.newBox).Methods("POST")
	return router
}
