package process

import (
	context "context"
	"fmt"
	"os"
	"os/exec"
	sync "sync"
	"syscall"

	"github.com/google/uuid"
	"github.com/kalliope-product/acsm-spark/internal/pb"
)

// A simple process-based session spawner, each session is a separate process running Spark Connect server
// For demo purpose only, use Kubernetes, Mesos or YARN for production.
type ProcessSessionSpawner struct {
	// Path to the spark connect command
	SparkConnectCmd string
	// Pid map to track spawned processes between SparkSession and OS process.
	pidMap map[uuid.UUID]struct {
		port int32
		pid  int
	}
	// mutex to protect pidMap as well as prevent simultaneous spawns
	mu sync.RWMutex
}

func NewProcessSessionSpawner(cmd string) *ProcessSessionSpawner {
	return &ProcessSessionSpawner{
		SparkConnectCmd: cmd,
		pidMap: make(map[uuid.UUID]struct {
			port int32
			pid  int
		}),
		mu: sync.RWMutex{},
	}
}

func (pss *ProcessSessionSpawner) getNextPort() int32 {
	// Try to find the minimum unused port starting from 15005
	// Normally it will be incrementing by 1 each time
	// But in some case, we have 15005, 15006, 15008 used, so next port should be 15007
	const startingPort int32 = 15005
	var seenPorts = map[int32]bool{}
	for _, entry := range pss.pidMap {
		seenPorts[entry.port] = true
	}
	newPort := startingPort
	for p := startingPort; p < 16000; p++ {
		if seen, ok := seenPorts[p]; !seen || !ok {
			return newPort
		}
	}
	// os check if port really free is not implemented yet
	return newPort
}

// Implement SessionSpawner interface methods for ProcessSessionSpawner here...
func (pss *ProcessSessionSpawner) SpawnSession(session *pb.SparkSession) (*pb.SparkSession, error) {
	// Implementation goes here...
	// We got the name and Id from the passed session object.
	// Get next port, starting from 15005
	pss.mu.Lock()
	defer pss.mu.Unlock()
	port := pss.getNextPort()
	// Build the command to spawn the Spark Connect server process
	cmd := exec.Command(
		pss.SparkConnectCmd,
		"--conf",
		fmt.Sprintf("spark.connect.grpc.binding.port=%d", port),
	)
	// Set environment variables if needed
	cmd.Env = os.Environ()
	// Start the process
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	// Store the pid and port in pidMap
	pss.pidMap[session.Id] = struct {
		port int32
		pid  int
	}{
		port: port,
		pid:  int(cmd.Process.Pid),
	}
	session.ConnectUrl = fmt.Sprintf("localhost:%d", port)
	return session, nil
}

func (pss *ProcessSessionSpawner) CheckSessionAlive(session *pb.SparkSession) bool {
	// Implementation goes here...
	pss.mu.Lock()
	defer pss.mu.Unlock()
	entry, exists := pss.pidMap[session.Id]
	if exists {
		// Check if process is still running
		process, err := os.FindProcess(entry.pid)
		if err == nil {
			// On Unix systems, sending signal 0 does not kill the process,
			// but will return an error if the process does not exist.
			err = process.Signal(syscall.Signal(0x0))
			if err == nil {
				return true
			}
		}
	}
	return false
}

func (pss *ProcessSessionSpawner) KillSession(session *pb.SparkSession) error {
	// Kill the session process
	pss.mu.Lock()
	defer pss.mu.Unlock()
	entry, exists := pss.pidMap[session.Id]
	if exists {
		process, err := os.FindProcess(entry.pid)
		if err == nil {
			err = process.Signal(os.Interrupt)
			if err != nil {
				return err
			}
			// Remove from pidMap
			delete(pss.pidMap, session.Id)
			return nil
		} else {
			return err
		}
	}
	return nil
}

func (pss *ProcessSessionSpawner) GetSessionConnectUrl(session *pb.SparkSession) (string, error) {
	// Implementation goes here...
	pss.mu.RLock()
	defer pss.mu.RUnlock()
	entry, exists := pss.pidMap[session.Id]
	if !exists {
		return "", fmt.Errorf("session not found")
	}
	return fmt.Sprintf("localhost:%d", entry.port), nil
}

func (pss *ProcessSessionSpawner) Start(ctx context.Context) error {
	// Implementation goes here...
	return nil
}

func (pss *ProcessSessionSpawner) Stop(ctx context.Context) error {
	// Implementation goes here...
	return nil
}
