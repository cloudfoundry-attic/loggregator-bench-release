package doppler_test

import (
	"fmt"
	"log"
	"testing"
)

func BenchmarkDopplerLatencyV1ToV1_100(b *testing.B) {
	log.Printf("Starting Doppler latency benchmark (V1ToV1_100) (b.N = %d)...", b.N)
	defer log.Printf("Done with Doppler latency benchmark (V1ToV1_100) (b.N = %d).", b.N)
	producer := newV1Producer(grpcConfig)
	consumer := newV1Consumer(grpcConfig)
	defer producer.closeSend()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			msg := fmt.Sprintf("message-%d", i)
			payload := buildPayload([]byte(msg))
			producer.send(payload)
		}
		// Assume Doppler drops some envelopes.
		consumer.observe(80)
	}
	b.StopTimer()
}

func BenchmarkDopplerLatencyV1ToV1_1(b *testing.B) {
	log.Printf("Starting Doppler latency benchmark (V1ToV1_1) (b.N = %d)...", b.N)
	defer log.Printf("Done with Doppler latency benchmark (V1ToV1_1) (b.N = %d).", b.N)
	producer := newV1Producer(grpcConfig)
	consumer := newV1Consumer(grpcConfig)
	defer producer.closeSend()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := fmt.Sprintf("message-%d", i)
		payload := buildPayload([]byte(msg))
		producer.send(payload)
		consumer.observe(1)
	}
	b.StopTimer()
}
