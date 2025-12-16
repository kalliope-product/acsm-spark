package proxy

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/kalliope-product/acsm-spark/internal/manager"
	"github.com/kalliope-product/acsm-spark/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type StreamDirector func(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error)

type ACSMServer struct {
	logger         hclog.Logger
	opts           *ACSMServerOpts
	sessionManager *manager.SessionManager
	pb.UnimplementedACSManagerServer
}

type ACSMServerOpts struct {
	Port int // Assume listening on 0.0.0.0
	// Static Proxy for now
	StaticProxyAddr string
	// Add more options here
	GrpcOpts []grpc.ServerOption
}

func NewACSMServer(opts *ACSMServerOpts, logger hclog.Logger) ACSMServer {
	return ACSMServer{
		logger: logger,
		opts:   opts,
		sessionManager: manager.NewSessionManager(
			logger,
			manager.WithProcessSpawner("./tests/spark_connect_server.sh"),
		),
	}
}

func (s *ACSMServer) Start(ctx context.Context) error {
	listeningAddr := fmt.Sprintf("0.0.0.0:%d", s.opts.Port)
	s.logger.Info("Starting Spark Connect Proxy ", "address", listeningAddr)
	opts := s.opts.GrpcOpts
	opts = append(opts, grpc.UnknownServiceHandler(TransparentHandler(s.proxyDirector(), s.logger)))
	srv := grpc.NewServer(opts...)
	pb.RegisterACSManagerServer(srv, s)
	serverCtx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	// Session Manager
	wg := sync.WaitGroup{}
	wg.Go(func() {
		s.sessionManager.Start(serverCtx)
	})
	// Termnation handling
	go func() {
		<-serverCtx.Done()
		s.logger.Info("Stopping Spark Connect Proxy")
		srv.GracefulStop()
		s.logger.Info("Waiting for all background processes to finish")
		wg.Wait()
		os.Exit(0)
	}()
	// Listener main
	listener, err := net.Listen("tcp", listeningAddr)
	if err != nil {
		return err
	}
	return srv.Serve(listener)
}

func (s *ACSMServer) proxyDirector() StreamDirector {
	return func(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		// If the request contain a session id, so we can route.
		if md.Get("session-id") == nil || len(md.Get("session-id")) == 0 {
			return nil, nil, fmt.Errorf("missing session-id in metadata")
		}
		if len(md.Get("session-id")) > 1 {
			return nil, nil, fmt.Errorf("multiple session-id in metadata")
		}
		sessionID := md.Get("session-id")[0]
		sessionUUID, err := uuid.Parse(sessionID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid session-id in metadata: %v", err)
		}
		session := s.sessionManager.GetSession(sessionUUID)
		if session == nil {
			return nil, nil, fmt.Errorf("session not found: %s", sessionID)
		}
		// Inspect the metadata
		// for k, v := range md {
		// 	s.logger.Debug("fullMethodName: %s Metadata key: %s, value: %v\n", fullMethodName, k, v)
		// }
		s.logger.Debug("Current metadata ", "md", md)
		ctx = metadata.NewOutgoingContext(ctx, md.Copy())
		return ctx, session.Connection, nil
	}
}

// DefaultDirector returns a very simple forwarding StreamDirector that forwards all
// calls.
// func DefaultDirector(cc grpc.ClientConnInterface) StreamDirector {
// 	return func(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error) {
// 		md, _ := metadata.FromIncomingContext(ctx)
// 		// Inspect the metadata
// 		for k, v := range md {
// 			fmt.Printf("fullMethodName: %s Metadata key: %s, value: %v\n", fullMethodName, k, v)
// 		}
// 		ctx = metadata.NewOutgoingContext(ctx, md.Copy())
// 		return ctx, cc, nil
// 	}
// }

func (s *ACSMServer) GetOrCreateSession(ctx context.Context, request *pb.ConnectSessionRequest) (*pb.ConnectSession, error) {
	s.logger.Info("GetOrCreateSession called", "session_name", request.SessionName)
	session, err := s.sessionManager.GetOrCreateSession(request.SessionName)
	if err != nil {
		return nil, err
	}
	return &pb.ConnectSession{
		SessionId:   session.Id.String(),
		SessionName: session.Name,
	}, nil
}

func (s *ACSMServer) ListSessions(ctx context.Context, request *pb.ConnectSessionListRequest) (*pb.ConnectSessionListResponse, error) {
	s.logger.Info("ListSessions called")
	sessions := s.sessionManager.ListSessions()
	var pbSessions []*pb.ConnectSession
	for _, session := range sessions {
		pbSessions = append(pbSessions, &pb.ConnectSession{
			SessionId:   session.Id.String(),
			SessionName: session.Name,
		})
	}
	return &pb.ConnectSessionListResponse{
		Sessions: pbSessions,
	}, nil
}
