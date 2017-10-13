package doppler_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.cloudfoundry.org/loggregator/doppler/app"
	"code.cloudfoundry.org/loggregator/plumbing"
	loggregator_v2 "code.cloudfoundry.org/loggregator/plumbing/v2"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
)

func BenchmarkDopplerThroughputV1ToV1(b *testing.B) {
	b.ReportAllocs()

	cleanup := saturateV1Ingress(grpcConfig)
	defer cleanup()
	consumer := newV1Consumer(grpcConfig)
	time.Sleep(5 * time.Second)

	b.ResetTimer()
	consumer.observe(b.N)
	b.StopTimer() // We don't want to measure the cleanup
}

func BenchmarkDopplerThroughputV2ToV1(b *testing.B) {
	b.ReportAllocs()

	cleanup := saturateV2Ingress(grpcConfig)
	defer cleanup()
	consumer := newV1Consumer(grpcConfig)
	time.Sleep(5 * time.Second)

	b.ResetTimer()
	consumer.observe(b.N)
	b.StopTimer() // We don't want to measure the cleanup
}

func BenchmarkDopplerThroughputV2ToV2(b *testing.B) {
	b.ReportAllocs()

	cleanup := saturateV2Ingress(grpcConfig)
	defer cleanup()
	consumer := newV2Consumer(grpcConfig)
	time.Sleep(5 * time.Second)

	b.ResetTimer()
	consumer.observe(b.N)
	b.StopTimer() // We don't want to measure the cleanup
}

func saturateV1Ingress(g app.GRPC) func() {
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

	client := plumbing.NewDopplerIngestorClient(conn)
	randEnvelopeData := randEnvelopeDataGen()

	go func() {
		defer wg.Done()
		var pusher plumbing.DopplerIngestor_PusherClient
		for atomic.LoadInt64(&done) == 0 {
			if pusher == nil {
				var err error
				ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
				pusher, err = client.Pusher(ctx)
				if err != nil {
					log.Println(err)
					time.Sleep(50 * time.Millisecond)
					continue
				}
			}

			pusher.Send(randEnvelopeData())
		}
	}()

	return func() {
		atomic.AddInt64(&done, 1)
		wg.Wait()
	}
}

func saturateV2Ingress(g app.GRPC) func() {
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

func randEnvelopeDataGen() func() *plumbing.EnvelopeData {
	var (
		envelopes []*plumbing.EnvelopeData
		i         int
	)

	for j := 0; j < 5; j++ {
		envelopes = append(envelopes, &plumbing.EnvelopeData{
			Payload: buildPayload(),
		})
	}

	return func() *plumbing.EnvelopeData {
		i++
		return envelopes[i%len(envelopes)]
	}
}

func buildPayload() []byte {
	buf := make([]byte, 10)
	rand.Read(buf)
	envelope := buildV1Log(buf)
	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

func buildV1Log(b []byte) *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String("doppler"),
		EventType: events.Envelope_LogMessage.Enum(),
		Timestamp: proto.Int64(time.Now().UnixNano()),
		LogMessage: &events.LogMessage{
			Message:     b,
			MessageType: events.LogMessage_OUT.Enum(),
			Timestamp:   proto.Int64(time.Now().UnixNano()),
		},
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
		batch = append(batch, buildV2Log(fmt.Sprintf("%d", i%20000), buf))
	}
	return batch
}

func buildV2Log(appID string, data []byte) *loggregator_v2.Envelope {
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
