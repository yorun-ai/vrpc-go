// Command portal invokes one vRPC method through a configured Portal endpoint.
package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
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
	clientName := flag.String("client-name", "", "Client application name")
	clientVersion := flag.String("client-version", "", "Client semantic version")
	clientInstanceID := flag.String("client-instance-id", "", "Client instance UUID")
	flag.Parse()
	authorization := map[string]string{}
	if key := os.Getenv("PORTAL_KEY"); key != "" {
		authorization["key"] = key
	}

	client, err := vrpc.NewClient(vrpc.Option{
		Identity: vrpc.Identity{
			Name:       *clientName,
			Version:    *clientVersion,
			InstanceID: *clientInstanceID,
		},
		Endpoint:      *endpoint,
		Timeout:       *timeout,
		Authorization: authorization,
	})
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	result, response, err := client.InvokeRaw(ctx, *service, *method, jsontext.Value(*params))
	if response != nil {
		fmt.Fprintf(os.Stderr, "HTTP=%d status=%s\n", response.HTTPStatus, response.Status)
	}
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}
