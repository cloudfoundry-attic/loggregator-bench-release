package latency_test

import (
	"log"
	"os"
	"strconv"
	"testing"

	dopplerApp "loggregator-release/src/code.cloudfoundry.org/loggregator/doppler/app"
)

var (
	d          *dopplerApp.Doppler
	grpcConfig dopplerApp.GRPC
)

func init() {
	spawnDoppler()
	spawnMetron()
	spawnTrafficController()
	spawnReverseLogProxy()
}

func benchmarkLatency(inVersion, outVersion int, rate string, b *testing.B) {
	b.ReportAllocs()

	// slow prime system
	b.ResetTimer()
	// emit single message
	// observe it come out
	b.StopTimer()
}

func BenchmarkLowThroughputLatencyV1ToV1(b *testing.B) { benchmarkLatench(1, 1, "low") }

// func BenchmarkHighThroughputLatencyV1ToV1(b *testing.B) { benchmarkLatench(1, 1, "high") }
// func BenchmarkLowThroughputLatencyV1ToV2(b *testing.B)  { benchmarkLatench(1, 2, "low") }
// func BenchmarkHighThroughputLatencyV1ToV2(b *testing.B) { benchmarkLatench(1, 2, "high") }
// func BenchmarkLowThroughputLatencyV2ToV1(b *testing.B)  { benchmarkLatench(2, 1, "low") }
// func BenchmarkHighThroughputLatencyV2ToV1(b *testing.B) { benchmarkLatench(2, 1, "high") }
// func BenchmarkLowThroughputLatencyV2ToV2(b *testing.B)  { benchmarkLatench(2, 2, "low") }
// func BenchmarkHighThroughputLatencyV2ToV2(b *testing.B) { benchmarkLatench(2, 2, "high") }

func spawnDoppler() {
	port, err := strconv.Atoi(os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Fatal(err)
	}

	grpcConfig = dopplerApp.GRPC{
		Port:     uint16(port),
		CertFile: os.Getenv("GRPC_CERT"),
		KeyFile:  os.Getenv("GRPC_KEY"),
		CAFile:   os.Getenv("GRPC_CA"),
	}

	d = dopplerApp.NewDoppler(grpcConfig)
	d.Start()
}

func spawnMetron() {
	// gaugeMap := stubGaugeMap()

	// promRegistry := prometheus.NewRegistry()
	// he := healthendpoint.New(promRegistry, gaugeMap)
	// clientCreds, err := plumbing.NewClientCredentials(
	// 	testservers.Cert("metron.crt"),
	// 	testservers.Cert("metron.key"),
	// 	testservers.Cert("loggregator-ca.crt"),
	// 	"doppler",
	// )
	// Expect(err).ToNot(HaveOccurred())

	// config := testservers.BuildMetronConfig("localhost", 1234)
	// config.Zone = "something-bad"
	// expectedHost, _, err := net.SplitHostPort(config.DopplerAddrWithAZ)
	// Expect(err).ToNot(HaveOccurred())

	// metronApp := metronApp.NewV1App(
	// 	&config,
	// 	he,
	// 	clientCreds,
	// 	spyMetricClient{},
	// )
	// go metronApp.Start()
}

func spawnTrafficController() {}
func spawnReverseLogProxy()   {}
