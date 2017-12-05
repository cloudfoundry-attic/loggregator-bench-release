package doppler_test

import (
	"log"
	"os"
	"strconv"
	"testing"

	"code.cloudfoundry.org/loggregator/router/app"
)

var (
	r          *app.Router
	grpcConfig app.GRPC
)

func TestMain(m *testing.M) {
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

	r = app.NewRouter(grpcConfig)
	r.Start()

	os.Exit(m.Run())
}
