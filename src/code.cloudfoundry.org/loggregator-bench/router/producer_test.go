package router_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"code.cloudfoundry.org/loggregator/plumbing"
	"code.cloudfoundry.org/loggregator/router/app"
	"google.golang.org/grpc"
)

type v1Producer struct {
	client plumbing.DopplerIngestor_PusherClient
}

func newV1Producer(g app.GRPC) *v1Producer {
	c := &v1Producer{}

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

	client := plumbing.NewDopplerIngestorClient(conn)
	for {
		if c.client == nil {
			var err error
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			c.client, err = client.Pusher(ctx)
			if err == nil {
				break
			}
			log.Println(err)
			time.Sleep(50 * time.Millisecond)
		}
	}
	return c
}

func (p *v1Producer) send(payload []byte) {
	err := p.client.Send(&plumbing.EnvelopeData{
		Payload: payload,
	})
	if err != nil {
		log.Fatalf("failed to send message: %s", err)
	}
}

func (p *v1Producer) closeSend() {
	p.client.CloseSend()
}
