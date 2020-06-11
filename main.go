package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
)

var (
	listenFlag  = flag.String("listen", ":8080", "address and port to listen")
	textFlag    = flag.String("text", "", "text to put on the webpage")
	versionFlag = flag.Bool("version", false, "display version information")

	stdoutW = os.Stdout
	stderrW = os.Stderr
)

func readinessProbe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK\n"))
}

func livenessProbe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK\n"))
}

func handlerPing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func dumpRequest(w http.ResponseWriter, r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Fprint(w, err.Error())
	} else {
		fmt.Fprint(w, string(requestDump))
	}
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	// get client IP address
	ip, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Print("userIP: [", r.RemoteAddr, "] is not IP:port")
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		log.Print("userIP: [", r.RemoteAddr, "] is not IP:port")
		return
	}

	fmt.Fprintf(w, "ClientIP: %s\n", ip)
	fmt.Fprintf(w, "ClientPort: %s\n", port)
	// The user could acccess the web server via a proxy or load balancer.
	// The above IP address will be the IP address of the proxy or load balancer
	// and not the user's machine. Let's read the r.header "X-Forwarded-For (XFF)".
	// In our example the value returned is nil, we consider there is no proxy,
	// the IP indicates the user's address.
	// WARNING: this header is optional and will only be defined when site is
	// accessed via non-anonymous proxy and takes precedence over RemoteAddr.
	// (read https://tools.ietf.org/html/rfc7239 before any further use).
	// proxied := r.Header.Get("X-FORWARDED-FOR")
	// if proxied != "" {
	// 	fmt.Fprintf(w, "X-FORWARDED-FOR: %s\n", proxied)
	// }
	host, err := os.Hostname()
	if err == nil {
		fmt.Fprintf(w, "HostFQDN %s\n", host)
	} else {
		fmt.Fprintf(w, "HostFQDN: %s\n ", err.Error())
	}
	fmt.Fprintf(w, "HostIP: %s\n", r.Host)
	fmt.Fprintf(w, "Protocol: %s\n", r.Proto)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "URL: %s\n", r.URL)

	fmt.Fprintln(w, "")
	for key, values := range r.Header {
		for _, value := range values {
			fmt.Fprintf(w, "%s: %s\n", key, value)
		}
	}

	fmt.Fprintln(w, "")
	if podName := os.Getenv("POD_NAME"); podName != "" {
		fmt.Fprintf(w, "PodName: %s\n", podName)
	}
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		fmt.Fprintf(w, "nodeName: %s\n", nodeName)
	}
	if podNamespace := os.Getenv("POD_NAMESPACE"); podNamespace != "" {
		fmt.Fprintf(w, "podNamespace: %s\n", podNamespace)
	}
	if podIP := os.Getenv("POD_IP"); podIP != "" {
		fmt.Fprintf(w, "podIP: %s\n", podIP)
	}
	if serviceAccountName := os.Getenv("SERVICE_ACCOUNT"); serviceAccountName != "" {
		fmt.Fprintf(w, "serviceAccountName: %s\n", serviceAccountName)
	}

}

func routeHandler(w http.ResponseWriter, r *http.Request) {
	podName := os.Getenv("POD_NAME")
	nodeName := os.Getenv("NODE_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	podIP := os.Getenv("POD_IP")
	serviceAccountName := os.Getenv("SERVICE_ACCOUNT")
	fmt.Fprintf(w, "%s,%s,%s,%s,%s\n", podName, nodeName, podNamespace, podIP, serviceAccountName)
}

func main() {

	// port := flag.String("port", ":8080", "port to serve on")
	flag.Parse()

	tlsCert := ""
	if tlscertFromEnv := os.Getenv("TLS_CERT"); tlscertFromEnv != "" {
		tlsCert = tlscertFromEnv
	}

	tlsKey := ""
	if tlskeyFromEnv := os.Getenv("TLS_CERT"); tlskeyFromEnv != "" {
		tlsKey = tlskeyFromEnv
	}

	server := http.NewServeMux()
	server.HandleFunc("/", indexPage)
	server.HandleFunc("/ready", readinessProbe)
	server.HandleFunc("/alive", livenessProbe)
	server.HandleFunc("/ping", handlerPing)
	server.HandleFunc("/dump", dumpRequest)
	server.HandleFunc("/route", routeHandler)

	log.Printf("Listening on port: %s ...\n", *listenFlag)
	if tlsCert != "" && tlsKey != "" {
		err := http.ListenAndServeTLS(*listenFlag, tlsCert, tlsKey, server)
		log.Fatal(err)
	} else {
		err := http.ListenAndServe(*listenFlag, server)
		log.Fatal(err)
	}
}
