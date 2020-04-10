package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"admission-mutate-server-sidecar-injector/admissionreview"
	"admission-mutate-server-sidecar-injector/kubectlient"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	client *kubernetes.Clientset
)

func main() {
	// Flags for local run
	port := flag.String("port", "4443", "port - default 4443")
	kubeconfig := flag.String("kc", "", "Path to a kubeconfig file")
	flag.Set("logtostderr", "true")
	flag.Parse()

	// Create k8s api client
	client = kubectlient.GetK8sCilent(*kubeconfig)

	// Test conction
	nsList, _ := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	log.Println(nsList)

	// Configure https server
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", handleMutate(client))

	s := &http.Server{
		Addr:           ":" + *port,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1048576
	}

	log.Fatal(s.ListenAndServeTLS("../ssl/mutateme-server.pem", "../ssl/mutateme-server.key"))
}

// https://stackoverflow.com/questions/33646948/go-using-mux-router-how-to-pass-my-db-to-my-handlers
func handleMutate(client *kubernetes.Clientset) http.HandlerFunc {
	handlerFunc := func(w http.ResponseWriter, request *http.Request) {
		// read the body / request
		reqBody, err := ioutil.ReadAll(request.Body)
		defer request.Body.Close()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
		}

		// mutate the request
		mutated, err := admissionreview.Mutate(reqBody, *client)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
		}

		// and write it back
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mutated)
	}
	return handlerFunc
}