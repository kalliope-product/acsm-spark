package manager

import (
	"context"
	"fmt"
	sync "sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/kalliope-product/acsm-spark/internal/manager/process"
	"github.com/kalliope-product/acsm-spark/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Idle Duration before a session is considered for cleanup
const SESSION_IDLE_DURATION = 30 * time.Minute
const SESSION_IDLE_CHECK_THRESHOLD = 5 * time.Minute // If no request in this duration, consider them idle
const SESSION_IDLE_CHECK_INTERVAL = 30 * time.Second // Check every 30 seconds

// Manage session created by managers

type SessionManager struct {
	sessions map[uuid.UUID]*pb.SparkSession // Shared map, so mutex follow
	logger   hclog.Logger
	spawner  SessionSpawner
	rwlock   sync.RWMutex
}

type SessionSpawner interface {
	// Spawn a new session
	SpawnSession(session *pb.SparkSession) (*pb.SparkSession, error)
	// Check if the session is still alive
	CheckSessionAlive(session *pb.SparkSession) bool
	// Kill the session if possible
	KillSession(session *pb.SparkSession) error
	// Get connection URL for the session
	GetSessionConnectUrl(session *pb.SparkSession) (string, error)
	// Start the spawner
	Start(ctx context.Context) error
	// Stop the spawner
	Stop(ctx context.Context) error
}

type SessionManagerOpt func(*SessionManager)

func WithProcessSpawner(cmd string) SessionManagerOpt {
	return func(sm *SessionManager) {
		sm.spawner = process.NewProcessSessionSpawner(cmd)
	}
}

func NewSessionManager(logger hclog.Logger, opts ...SessionManagerOpt) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[uuid.UUID]*pb.SparkSession), // Local storage for sessions, in the case of distributed, we may need a shared storage, something like redis or just plain old postgres
		spawner:  nil,                                  // To be set by options
		rwlock:   sync.RWMutex{},
		logger:   logger,
	}
	for _, opt := range opts {
		opt(sm)
	}
	return sm
}

// Main start function for session manager
func (sm *SessionManager) Start(ctx context.Context) error {
	sm.logger.Info("Starting Session Manager")
	if sm.spawner == nil {
		return fmt.Errorf("no session spawner configured")
	}
	wg := sync.WaitGroup{}
	// Start the spawner in the context
	wg.Go(func() {
		sm.spawner.Start(ctx)
	})
	// Start the cleanup loop
	wg.Go(func() {
		sm.cleanupLoop(ctx)
	})
	sm.logger.Info("Session Manager started and waiting for requests")
	// Waiting for the context to be cancelled
	<-ctx.Done()
	wg.Wait()
	sm.logger.Info("Stopping Session Manager")
	// Stop the spawner
	return sm.spawner.Stop(ctx)
}

// Get or Create idle session
func (sm *SessionManager) GetOrCreateSession(name string) (*pb.SparkSession, error) {
	sm.rwlock.Lock()
	defer sm.rwlock.Unlock()
	// Check for existing idle session
	for _, session := range sm.sessions {
		// We prefer reusing idle sessions with the same name
		// If not just get the first idle session
		// This behavior can be changed later, depending on requirements
		if session.Name == name && session.IsIdle {
			session.IsIdle = false
			session.LastRequest = time.Now()
			return session, nil
		}
	}
	// Loop again check for idle session.
	for _, session := range sm.sessions {
		if session.IsIdle {
			session.Name = name // Update the name.
			session.IsIdle = false
			session.LastRequest = time.Now()
			return session, nil
		}
	}
	// Create new session
	newSession := &pb.SparkSession{
		Name: name,
		Id:   uuid.Must(uuid.NewV7()),
	}
	// Spawn the session using spawner
	spawnedSession, err := sm.spawner.SpawnSession(newSession)
	if err != nil {
		return nil, err
	}
	spawnedSession.IsIdle = false
	spawnedSession.LastRequest = time.Now()
	// Try to connect to the session to verify
	spawnedSession.Connection, err = grpc.NewClient(spawnedSession.ConnectUrl, grpc.WithTransportCredentials(
		// Insecure for now
		insecure.NewCredentials(),
	))

	if err != nil {
		return nil, err
	}
	// Store in sessions map
	sm.sessions[newSession.Id] = spawnedSession
	return spawnedSession, nil
}

func (sm *SessionManager) ListSessions() []*pb.SparkSession {
	sm.rwlock.RLock()
	defer sm.rwlock.RUnlock()
	var sessions []*pb.SparkSession
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

func (sm *SessionManager) GetSession(id uuid.UUID) *pb.SparkSession {
	sm.rwlock.RLock()
	defer sm.rwlock.RUnlock()
	if session, ok := sm.sessions[id]; ok {
		return session
	}
	return nil
}

// Cleanup loop to mark idle sessions, terminate them after idle duration
func (sm *SessionManager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(SESSION_IDLE_CHECK_INTERVAL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Encapsulate in a function to ensure locks are released
			func() {
				sm.logger.Debug("Running session cleanup loop", "total_sessions", len(sm.sessions))
				sm.rwlock.Lock()
				defer sm.rwlock.Unlock()
				// Hopefully we managed small number of sessions, otherwise we may lock the page for too long.
				// The other way is to use sync.Map but that is overkill for now.
				for id, session := range sm.sessions {
					if !session.IsIdle && time.Since(session.LastRequest) > SESSION_IDLE_CHECK_THRESHOLD {
						// Mark as idle
						session.IsIdle = true
					} else if session.IsIdle && time.Since(session.LastRequest) > SESSION_IDLE_DURATION {
						// Terminate the session
						err := sm.spawner.KillSession(session)
						if err != nil {
							sm.logger.Error("Failed to kill idle session", "session_id", session.Id.String(), "error", err)
							continue
						}
						sm.logger.Info("Killed idle session", "session_id", session.Id.String())
						// Remove from sessions map
						delete(sm.sessions, id)
					}
				}
			}()
		}
	}
}
