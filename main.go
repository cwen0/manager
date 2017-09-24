package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/GregoryIan/agent/util"
	"github.com/coreos/etcd/clientv3"
	"github.com/cwen0/manager/api"
	"github.com/ngaut/log"
	"github.com/pingcap/schrodinger/cluster"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	endpoint             string
	kubeconfig           string
	tidbCloudManagerAddr string
	logFile              string
	logLevel             string
	pprofAddr            string
	addr                 string
	printVersion         bool
)

func init() {
	flag.StringVar(&pprofAddr, "pprof", "0.0.0.0:10080", "manager pprof address")
	flag.StringVar(&addr, "addr", "0.0.0.0:3000", "manager listen address")
	flag.StringVar(&tidbCloudManagerAddr, "cloud-manager-addr", "localhost:32333", "tidb cloud manager address")
	flag.StringVar(&logFile, "log-file", "", "manager log file")
	flag.StringVar(&logLevel, "log-level", "info", "manager log level: info, warn, fatal, error")
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.StringVar(&endpoint, "endpoint", "https://localhost:2379", "etcd endpoint")
	if home := homeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
}

func main() {
	flag.Parse()
	if printVersion {
		util.PrintInfo()
		os.Exit(0)
	}
	initLogger()
	go func() {
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Fatal(err)
		}
	}()
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(strings.TrimSpace(endpoint), " "),
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Fatal(err)
	}
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	// creates the clientset
	k8sCli, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	clusterCli := cluster.NewK8sClient(tidbCloudManagerAddr)
	log.Infof("begin to listen %s", addr)
	m := api.NewManager(addr, etcdCli, k8sCli, clusterCli)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		m.Run()
	}()
	time.Sleep(time.Second)
	<-sigs
	log.Info("closing manager")
	m.Close()
	log.Info("manager is closed")
}

func initLogger() {
	log.SetLevelByString(logLevel)
	if len(logFile) > 0 {
		log.SetOutputByName(logFile)
		log.SetRotateByDay()
	}
}

//func main() {
//	boxer, err := box.New("manager-test", clusterCli, nil, 10*time.Minute, etcdCli, k8sCli)
//	if err != nil {
//		log.Fatal(err)
//	}
//	boxer.Start()
//	caseCfg := box.CaseConfig{
//		Name:       "bank",
//		BinaryName: "bank-test",
//		URL:        "ulcoud.cn/pingcap/bank.tar.gz",
//		Image:      "uhub.service.ucloud.cn/tidb/agent:",
//		Cluster: &types.Cluster{
//			Name:               "cluster-1",
//			TidbLease:          5,
//			MonitorReserveDays: 14,
//			Pd: &types.PodSpec{
//				Version: "9a91320",
//				Size:    1,
//			},
//			Tidb: &types.PodSpec{
//				Version: "b22e639",
//				Size:    1,
//			},
//			Tikv: &types.PodSpec{
//				Version: "f9244e0",
//				Size:    3,
//			},
//			Monitor: &types.PodSpec{
//				Version: "4.2.0,v1.5.2,v0.3.1",
//				Size:    1,
//			},
//		},
//	}
//	cs, err := boxer.AddCase(caseCfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := boxer.DeleteCase(cs); err != nil {
//		log.Fatal(err)
//	}
//	boxer.Stop()
//	boxer.State()
//}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
