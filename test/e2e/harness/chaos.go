//go:build chaos

package harness

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	toxiclient "github.com/Shopify/toxiproxy/v2/client"
	"github.com/testcontainers/testcontainers-go/modules/toxiproxy"

	"github.com/s4bb4t/infra-layer-practice/internal/infrastructure/contracttest"
	"github.com/s4bb4t/infra-layer-practice/internal/testinfra"
)

const ToxiproxyImage = "ghcr.io/shopify/toxiproxy:2.12.0"

// Toxic is a Postgres and a RabbitMQ behind Toxiproxy. Every byte between the
// service and them goes through a proxy you can slow down, cut or break:
//
//	env.PG.AddToxic("lag", "latency", "downstream", 1, toxiclient.Attributes{"latency": 2000})
//	env.Rabbit.Disable() // connection refused / reset
//	env.Rabbit.Enable()
type Toxic struct {
	PG, Rabbit *toxiclient.Proxy
	// PostgresDSN and RabbitURI point at the proxies, not at the containers.
	PostgresDSN, RabbitURI string
	Client                 *toxiclient.Client
}

// StartToxic starts the containers and the proxy once per test binary and
// registers "postgres-toxic" and "rabbit-toxic" for Build.
func StartToxic(t *testing.T) *Toxic {
	t.Helper()
	ctx := context.Background()
	inf := Shared()

	pgURI := inf.Postgres(t)
	rmqURI := inf.Rabbit(t)
	pg, _ := inf.Container("postgres")
	rmq, _ := inf.Container("rabbit")

	pgIP, err := pg.ContainerIP(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rmqIP, err := rmq.ContainerIP(ctx)
	if err != nil {
		t.Fatal(err)
	}

	proxyURI := inf.get(t, "toxiproxy", func(ctx context.Context) (*testinfra.Container, error) {
		c, err := toxiproxy.Run(ctx, ToxiproxyImage,
			toxiproxy.WithProxy("postgres", net.JoinHostPort(pgIP, "5432")),
			toxiproxy.WithProxy("rabbit", net.JoinHostPort(rmqIP, "5672")),
		)
		if err != nil {
			return nil, err
		}
		uri, err := c.URI(ctx)
		if err != nil {
			return nil, err
		}
		// Proxies get listen ports 8666, 8667, ... in WithProxy order.
		pgHost, pgPort, err := c.ProxiedEndpoint(8666)
		if err != nil {
			return nil, err
		}
		rmqHost, rmqPort, err := c.ProxiedEndpoint(8667)
		if err != nil {
			return nil, err
		}
		proxied[0] = net.JoinHostPort(pgHost, pgPort)
		proxied[1] = net.JoinHostPort(rmqHost, rmqPort)
		return &testinfra.Container{Container: c, URI: uri}, nil
	})

	client := toxiclient.NewClient(proxyURI)
	if err := client.ResetState(); err != nil { // no toxics left from a previous test
		t.Fatal(err)
	}
	pgProxy, err := client.Proxy("postgres")
	if err != nil {
		t.Fatal(err)
	}
	rmqProxy, err := client.Proxy("rabbit")
	if err != nil {
		t.Fatal(err)
	}

	env := &Toxic{
		PG: pgProxy, Rabbit: rmqProxy, Client: client,
		PostgresDSN: replaceHost(pgURI, proxied[0]),
		RabbitURI:   replaceHost(rmqURI, proxied[1]),
	}
	t.Cleanup(func() { _ = client.ResetState() })

	RegisterStore("postgres-toxic", func(t *testing.T, _ *Infra) contracttest.Store {
		return PostgresStoreAt(t, env.PostgresDSN)
	})
	RegisterBroker("rabbit-toxic", func(t *testing.T, _ *Infra, topic string) contracttest.Broker {
		return RabbitBrokerAt(t, env.RabbitURI, topic)
	})
	return env
}

var proxied [2]string

// replaceHost swaps host:port inside scheme://user:pass@host:port/rest.
func replaceHost(uri, hostport string) string {
	at := strings.LastIndex(uri, "@")
	rest := uri[at+1:]
	end := strings.IndexAny(rest, "/?")
	if end < 0 {
		end = len(rest)
	}
	return uri[:at+1] + hostport + rest[end:]
}

func (e *Toxic) String() string { return fmt.Sprintf("pg=%s rabbit=%s", e.PostgresDSN, e.RabbitURI) }
