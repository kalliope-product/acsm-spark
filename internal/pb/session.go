package pb

import (
	"time"

	"github.com/google/uuid"
	grpc "google.golang.org/grpc"
)

type SparkSession struct {
	Name        string
	Id          uuid.UUID
	ConnectUrl  string                   // How to connect to the session
	Connection  grpc.ClientConnInterface // We cache the client connection here
	IsIdle      bool                     // Whether the session is taken or idle
	LastRequest time.Time                // Last request time for idle session management
}
