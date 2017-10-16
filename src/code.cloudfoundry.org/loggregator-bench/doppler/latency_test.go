package doppler_test

import (
	"fmt"
	"testing"
)

func BenchmarkDopplerLatencyV1ToV1LowThroughput(b *testing.B) {
	producer := newV1Producer(grpcConfig)
	consumer := newV1Consumer(grpcConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := fmt.Sprintf("message-%d", i)
		payload := buildPayload([]byte(msg))
		producer.send(payload)
		consumer.waitFor(payload, b)
	}
}
