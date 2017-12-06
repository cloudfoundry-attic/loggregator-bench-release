package router_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.cloudfoundry.org/loggregator/plumbing"
	loggregator_v2 "code.cloudfoundry.org/loggregator/plumbing/v2"
	"code.cloudfoundry.org/loggregator/router/app"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
)

func BenchmarkRouterThroughputV1ToV1(b *testing.B) {
	log.Println("Starting Router throughput benchmark (V1ToV1)...")
	defer log.Println("Done with Router throughput benchmark (V1ToV1).")
	b.ReportAllocs()

	cleanup := saturateV1Ingress(grpcConfig)
	defer cleanup()
	consumer := newV1Consumer(grpcConfig)
	defer consumer.stop()

	b.ResetTimer()
	consumer.observe(b.N)
	b.StopTimer() // We don't want to measure the cleanup
}

func BenchmarkRouterThroughputV2ToV1(b *testing.B) {
	log.Println("Starting Router throughput benchmark (V2ToV1)...")
	defer log.Println("Done with Router throughput benchmark (V2ToV1).")
	b.ReportAllocs()

	cleanup := saturateV2Ingress(grpcConfig)
	defer cleanup()
	consumer := newV1Consumer(grpcConfig)
	defer consumer.stop()

	b.ResetTimer()
	consumer.observe(b.N)
	b.StopTimer() // We don't want to measure the cleanup
}

func BenchmarkRouterThroughputV2ToV2(b *testing.B) {
	log.Println("Starting Router throughput benchmark (V2ToV2)...")
	defer log.Println("Done with Router throughput benchmark (V2ToV2).")
	b.ReportAllocs()

	cleanup := saturateV2Ingress(grpcConfig)
	defer cleanup()
	consumer := newV2Consumer(grpcConfig)
	defer consumer.stop()

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
			Payload: buildPayload(buildRandomMsg()),
		})
	}

	return func() *plumbing.EnvelopeData {
		i++
		return envelopes[i%len(envelopes)]
	}
}

func buildRandomMsg() []byte {
	buf := make([]byte, 10)
	rand.Read(buf)
	return buf
}

func buildPayload(msg []byte) []byte {
	envelope := buildV1Log(msg)
	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

func buildV1Log(b []byte) *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String("router"),
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
