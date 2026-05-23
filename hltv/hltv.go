package hltv

import (
	"HLTV-Manager/docker"
	log "HLTV-Manager/logger"
	"fmt"
	"time"
	"sync"
)

type HLTV struct {
	ID       int
	Settings Settings
	Demos    []Demos
	Docker   *docker.Docker
	Parser   Parser
	isRunning bool
	quitChan  chan struct{}

	logBuffer   *LogBuffer
	subscribers map[chan string]struct{}
	mu          sync.Mutex
}

type Settings struct {
	Name             string
	ShowIP           string
	Connect          string
	Port             string
	GameID           string
	DemoDir          string
	DemoName         string
	MaxDemoDay       string
	DebugTerminalLog bool
	Cvars            []string
}

type Demos struct {
	ID   int
	Name string
	Date string
	Time string
	Map  string
}

func NewHLTV(id int, settings Settings) (*HLTV, error) {
	docker, err := docker.NewDockerClient()
	if err != nil {
		log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Error creating Docker client: %v", id, settings.Name, err)
		return nil, err
	}

	return &HLTV{
		ID:       id,
		Settings: settings,
		Docker:   docker,
	}, nil
}

func (hltv *HLTV) Start() error {
	var err error

	hltv.Settings.DemoDir, err = createDemosDir(hltv)
	if err != nil {
		return err
	}

	cfgPath, err := createHltvCfg(hltv)
	if err != nil {
		return err
	}

	hltvData := docker.Hltv{
		ID:     hltv.ID,
		Name:   hltv.Settings.Name,
		GameID: hltv.Settings.GameID,
	}

	err = hltv.Docker.CreateAndStart(docker.HltvContainerConfig{
		Cmd: []string{
			"+connect", hltv.Settings.Connect,
			"-port", hltv.Settings.Port,
			"+record", hltv.Settings.DemoName,
		},
		DemoPath: hltv.Settings.DemoDir,
		CfgPath:  cfgPath,
		Port:     hltv.Settings.Port,
		Hltv:     hltvData,
	})
	if err != nil {
		return err
	}

	hltv.quitChan = make(chan struct{})
	go hltv.TerminalControl()
	hltv.logBuffer = NewLogBuffer(200)
	hltv.subscribers = make(map[chan string]struct{})
	fmt.Println(hltv.Demos)
	hltv.isRunning = true

	return nil
}

func (hltv *HLTV) Restart() error {
	log.WarningLogger.Printf("HLTV (ID: %d, Name: %s) Restarting container...", hltv.ID, hltv.Settings.Name)

	if err := hltv.Quit(); err != nil {
		log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Error during quit: %v", hltv.ID, hltv.Settings.Name, err)
	}

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Failed to recreate Docker client: %v", hltv.ID, hltv.Settings.Name, err)
		return err
	}
	hltv.Docker = dockerClient

	if err := hltv.Start(); err != nil {
		log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Failed to restart: %v", hltv.ID, hltv.Settings.Name, err)
		return err
	}

	log.InfoLogger.Printf("HLTV (ID: %d, Name: %s) Restart completed successfully.", hltv.ID, hltv.Settings.Name)
	return nil
}

func (hltv *HLTV) Quit() error {
	err := hltv.WriteCommand("quit")
	if err != nil {
		log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Failed to write quit command: %v", hltv.ID, hltv.Settings.Name, err)

		if closer, ok := hltv.Docker.Attach.Conn.(interface{ CloseWrite() error }); ok {
            _ = closer.CloseWrite()
        }
        hltv.Docker.Attach.Close()
        hltv.isRunning = false
        return err
	}

	select {
    case <-hltv.quitChan:
        log.InfoLogger.Printf("HLTV (ID: %d, Name: %s) Demo archived, shutting down.", hltv.ID, hltv.Settings.Name)
    case <-time.After(10 * time.Second):
        log.WarningLogger.Printf("HLTV (ID: %d, Name: %s) Timeout waiting for demo completion, forcing close.", hltv.ID, hltv.Settings.Name)
    }

    if closer, ok := hltv.Docker.Attach.Conn.(interface{ CloseWrite() error }); ok {
        _ = closer.CloseWrite()
    }
    hltv.Docker.Attach.Close()
    hltv.isRunning = false
    return nil
}

func (hltv *HLTV) WriteCommand(cmd string) error {
	_, err := hltv.Docker.Attach.Conn.Write([]byte(cmd + "\n"))
	return err
}

func (hltv *HLTV) IsRunning() bool {
    return hltv.isRunning
}

type LogBuffer struct {
	lines []string
	size  int
	start int
	count int
	mu    sync.Mutex
}

func NewLogBuffer(size int) *LogBuffer {
	return &LogBuffer{
		lines: make([]string, size),
		size:  size,
	}
}

func (lb *LogBuffer) Add(line string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if lb.count < lb.size {
		lb.lines[lb.count] = line
		lb.count++
	} else {
		lb.lines[lb.start] = line
		lb.start = (lb.start + 1) % lb.size
	}
}

func (lb *LogBuffer) Snapshot() []string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	out := make([]string, lb.count)
	if lb.count < lb.size {
		copy(out, lb.lines[:lb.count])
	} else {
		for i := 0; i < lb.size; i++ {
			idx := (lb.start + i) % lb.size
			out[i] = lb.lines[idx]
		}
	}
	return out
}

func (hltv *HLTV) broadcastLine(line string) {
	if hltv.logBuffer != nil {
		hltv.logBuffer.Add(line)
	}
	hltv.mu.Lock()
	defer hltv.mu.Unlock()
	for ch := range hltv.subscribers {
		select {
		case ch <- line:
		default:
		}
	}
}

func (hltv *HLTV) Subscribe(ch chan string) {
	hltv.mu.Lock()
	defer hltv.mu.Unlock()
	hltv.subscribers[ch] = struct{}{}
}

func (hltv *HLTV) Unsubscribe(ch chan string) {
	hltv.mu.Lock()
	defer hltv.mu.Unlock()
	delete(hltv.subscribers, ch)
	close(ch)
}

func (hltv *HLTV) GetLogSnapshot() []string {
    if hltv.logBuffer == nil {
        return nil
    }
    return hltv.logBuffer.Snapshot()
}
/*

hltv_manager    | ***** FATAL ERROR *****
hltv_manager    | Server::SetState: not valid m_ServerState (6 -> 8).
hltv_manager    | *** STOPPING SYSTEM ***

*/
