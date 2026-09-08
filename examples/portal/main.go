// Command portal invokes one vRPC method through a configured Portal endpoint.
package main

import (
	"context"
	"encoding/json/jsontext"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"go.yorun.ai/vrpc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	endpoint := flag.String("endpoint", "", "Portal invocation URL including /invoke")
	service := flag.String("service", "", "fully qualified service name")
	method := flag.String("method", "", "method name")
	params := flag.String("params", "{}", "JSON argument object")
	timeout := flag.Duration("timeout", 30*time.Second, "call timeout")
	flag.Parse()
	client, err := vrpc.NewClient(vrpc.Options{Endpoint: *endpoint, Timeout: *timeout, Authorization: func(context.Context) (string, error) { return os.Getenv("PORTAL_AUTHORIZATION"), nil }})
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	var result jsontext.Value
	response, err := client.Call(ctx, *service, *method, jsontext.Value(*params), &result)
	if response != nil {
		fmt.Fprintf(os.Stderr, "HTTP=%d status=%s portal-trace-id=%s\n", response.HTTPStatus, response.Status, response.PortalTraceID)
	}
	if err != nil {
		return err
	}
	fmt.Println(string(result))
	return nil
}
