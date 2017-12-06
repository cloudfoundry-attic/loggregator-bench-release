package router_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"code.cloudfoundry.org/loggregator/plumbing"
	loggregator_v2 "code.cloudfoundry.org/loggregator/plumbing/v2"
	"code.cloudfoundry.org/loggregator/router/app"
	"google.golang.org/grpc"
)

type v1Consumer struct {
	client plumbing.Doppler_BatchSubscribeClient
	cancel func()
}

func newV1Consumer(g app.GRPC) *v1Consumer {
	c := &v1Consumer{}

	creds, err := plumbing.NewClientCredentials(
		g.CertFile,
		g.KeyFile,
		g.CAFile,
		"doppler",
	)

	if err != nil {
		log.Fatal(err)
	}

	h := fmt.Sprintf("localhost:%d", g.Port)
	log.Printf("v1Consumer dialing Doppler: %s", h)
	conn, err := grpc.Dial(
		h,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := plumbing.NewDopplerClient(conn)
	for {
		if c.client == nil {
			var err error
			ctx, cancel := context.WithCancel(context.Background())
			c.client, err = client.BatchSubscribe(ctx, &plumbing.SubscriptionRequest{})
			if err == nil {
				c.cancel = cancel
				break
			}
			log.Println(err)
			time.Sleep(50 * time.Millisecond)
		}
	}
	return c
}

func (c *v1Consumer) observe(n int) {
	var count int
	for {
		r, err := c.client.Recv()
		if err != nil {
			log.Panic(err)
		}
		count += len(r.Payload)
		if count >= n {
			break
		}
	}
}

func (c *v1Consumer) stop() {
	c.cancel()
}

func (c *v1Consumer) waitFor(want []byte, b *testing.B) {
	for i := 0; i < 100; i++ {
		resp, err := c.client.Recv()
		if err != nil {
			b.Fatalf("v1Consumer failed to receive: %s", err)
			return
		}
		for _, got := range resp.Payload {
			if equal(want, got) {
				return
			}
		}
	}

	b.Fatal("Did not receive the expected payload")
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		if b[i] != x {
			return false
		}
	}
	return true
}

type v2Consumer struct {
	client loggregator_v2.Egress_BatchedReceiverClient
	cancel func()
}

func newV2Consumer(g app.GRPC) *v2Consumer {
	c := &v2Consumer{}

	creds, err := plumbing.NewClientCredentials(
		g.CertFile,
		g.KeyFile,
		g.CAFile,
		"doppler",
	)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", g.Port), grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatal(err)
	}

	client := loggregator_v2.NewEgressClient(conn)
	for {
		if c.client == nil {
			var err error
			ctx, cancel := context.WithCancel(context.Background())
			c.client, err = client.BatchedReceiver(ctx, &loggregator_v2.EgressBatchRequest{})
			if err == nil {
				c.cancel = cancel
				break
			}
			log.Println(err)
			time.Sleep(50 * time.Millisecond)
		}
	}
	return c
}

func (c *v2Consumer) observe(n int) {
	var count int
	for {
		r, err := c.client.Recv()
		if err != nil {
			log.Panic(err)
		}
		count += len(r.Batch)
		if count > n {
			break
		}
	}
}

func (c *v2Consumer) stop() {
	c.cancel()
}
