package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/ngaut/log"
	"github.com/pingcap/schrodinger/box"
	"github.com/pingcap/schrodinger/cluster"
	"github.com/pingcap/schrodinger/cluster/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		panic(err.Error())
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	k8sCli, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	clusterCli := cluster.NewK8sClient("localhost:32333")
	boxer, err := box.New("master", clusterCli, nil, 10*time.Minute, etcdCli, k8sCli)
	if err != nil {
		log.Fatal(err)
	}
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		sig := <-sc
		log.Infof("Got signal [%d] to exit.", sig)
		boxer.Destroy()
	}()
	boxer.Start()
	caseCfg := box.CaseConfig{
		Name:       "bank",
		BinaryName: "bank-test",
		URL:        "ulcoud.cn/pingcap/bank.tar.gz",
		Image:      "uhub.service.ucloud.cn/pingcap/agent",
		Cluster: &types.Cluster{
			Name: "cluster_1",
		},
	}
	cs, err := boxer.AddCase(caseCfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := boxer.DeleteCase(cs); err != nil {
		log.Fatal(err)
	}
	boxer.Stop()
	boxer.State()
}
