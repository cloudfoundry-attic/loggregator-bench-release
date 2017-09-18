package doppler_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.cloudfoundry.org/loggregator/doppler/app"
	"code.cloudfoundry.org/loggregator/plumbing"
	loggregator_v2 "code.cloudfoundry.org/loggregator/plumbing/v2"
	"google.golang.org/grpc"
)

var (
	d          *app.Doppler
	grpcConfig app.GRPC
)

func init() {
	port, err := strconv.Atoi(os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Fatal(err)
	}

	grpcConfig = app.GRPC{
		Port:     uint16(port),
		CertFile: os.Getenv("GRPC_CERT"),
		KeyFile:  os.Getenv("GRPC_KEY"),
		CAFile:   os.Getenv("GRPC_CA"),
	}

	d = app.NewDoppler(grpcConfig)
	d.Start()
}

func BenchmarkDopplerThroughput(b *testing.B) {
	b.ReportAllocs()

	cleanup := saturate(grpcConfig)
	defer cleanup()
	consumer := newConsumer(grpcConfig)
	time.Sleep(5 * time.Second)

	b.ResetTimer()
	consumer.observe(b.N)
	b.StopTimer()
}

func saturate(g app.GRPC) func() {
	done := int64(0)
	var wg sync.WaitGroup
	wg.Add(1)

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

	client := loggregator_v2.NewDopplerIngressClient(conn)
	randBatch := randBatchGen()

	go func() {
		defer wg.Done()
		var sender loggregator_v2.DopplerIngress_BatchSenderClient
		for atomic.LoadInt64(&done) == 0 {
			if sender == nil {
				var err error
				ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
				sender, err = client.BatchSender(ctx)
				if err != nil {
					log.Println(err)
					time.Sleep(50 * time.Millisecond)
					continue
				}
			}

			sender.Send(randBatch())
		}
	}()

	return func() {
		atomic.AddInt64(&done, 1)
		wg.Wait()
	}
}

func randBatchGen() func() *loggregator_v2.EnvelopeBatch {
	var (
		batches []*loggregator_v2.EnvelopeBatch
		i       int
	)

	for j := 0; j < 5; j++ {
		batches = append(batches, &loggregator_v2.EnvelopeBatch{
			Batch: buildBatch(),
		})
	}

	return func() *loggregator_v2.EnvelopeBatch {
		i++
		return batches[i%len(batches)]
	}
}

func buildBatch() (batch []*loggregator_v2.Envelope) {
	for i := 0; i < 100; i++ {
		buf := make([]byte, 10)
		rand.Read(buf)
		batch = append(batch, buildLog(fmt.Sprintf("%d", i%20000), buf))
	}
	return batch
}

func buildLog(appID string, data []byte) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId:  appID,
		Timestamp: time.Now().UnixNano(),
		Message: &loggregator_v2.Envelope_Log{
			Log: &loggregator_v2.Log{
				Payload: data,
			},
		},
	}
}

type consumer struct {
	client plumbing.Doppler_BatchSubscribeClient
}

func newConsumer(g app.GRPC) *consumer {
	c := &consumer{}

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

func (c *consumer) observe(n int) {
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
