package doppler_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"code.cloudfoundry.org/loggregator/doppler/app"
	"code.cloudfoundry.org/loggregator/plumbing"
	loggregator_v2 "code.cloudfoundry.org/loggregator/plumbing/v2"
	"google.golang.org/grpc"
)

type v1Consumer struct {
	client plumbing.Doppler_BatchSubscribeClient
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

	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%d", g.Port),
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := plumbing.NewDopplerClient(conn)
	for {
		if c.client == nil {
			var err error
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			c.client, err = client.BatchSubscribe(ctx, &plumbing.SubscriptionRequest{})
			if err == nil {
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
		if count > n {
			break
		}
	}
}

type v2Consumer struct {
	client loggregator_v2.Egress_BatchedReceiverClient
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
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			c.client, err = client.BatchedReceiver(ctx, &loggregator_v2.EgressBatchRequest{})
			if err == nil {
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