package mobile

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"guarch/pkg/config"
	"guarch/pkg/core/dns"
	"guarch/pkg/core/sni"
	"guarch/pkg/cover"
	"guarch/pkg/engine"
	"guarch/pkg/mux"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

const (
	Version           = "1.0.1"
	MaxRetryAttempts  = 3
	RetryDelay        = 5 * time.Second
	ConnectionTimeout = 30 * time.Second
	HandshakeTimeout  = 30 * time.Second
)

var (
	goLogFile *os.File
	goLogMu   sync.Mutex
)

type Callback interface {
	OnStatusChanged(status string)
	OnStatsUpdate(jsonData string)
	OnLog(level, message string)
	OnError(errorMsg string)
	OnSNIRotation(newSNI string)
	OnDNSFallback(enabled bool)
}

type UserSettings struct {
	SocksPort        int `json:"socks_port"`
	DialTimeout      int `json:"dial_timeout"`
	HandshakeTimeout int `json:"handshake_timeout"`
}

type Engine struct {
	mu       sync.RWMutex
	callback Callback
	ctx      context.Context
	cancel   context.CancelFunc

	config       *config.ServerConfig
	userSettings *UserSettings

	muxConn      *mux.Mux
	groukSession *transport.GroukSession
	groukUDP     *net.UDPConn
	
	sniManager    *sni.Manager
	coverManager  *cover.Manager
	adaptiveCover *cover.AdaptiveCover
	dnsClient     *dns.Client
	connector     *engine.Connector

	listener net.Listener

	status           string
	stats            *engineStats
	protocol         string
	usingDNSFallback atomic.Bool
	batteryLevel     int
	dataSaverMode    bool
	retryCount       int
}

type engineStats struct {
	mu               sync.RWMutex
	totalUpload      int64
	totalDownload    int64
	coverRequests    int64
	activeStreams    int32
	totalConnections int64
	startTime        time.Time
	connectTime      time.Time

	currentSNI     string
	sniSwitches    int64
	dnsQueriesSent int64
	activityLevel  string
	lastSpeedUp    int64
	lastSpeedDown  int64
}

func InitGoLog(logPath string) {
	goLogMu.Lock()
	defer goLogMu.Unlock()
	
	if goLogFile != nil {
		log.Println("[Go] Log already initialized")
		return
	}
	
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[Go] Failed to open log file: %v", err)
		return
	}
	
	goLogFile = f
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	log.Println("[Go] ====== ENGINE LOG STARTED ======")
	log.Printf("[Go] Version: %s", Version)
	log.Printf("[Go] Log file: %s", logPath)
}

func CloseGoLog() {
	goLogMu.Lock()
	defer goLogMu.Unlock()
	
	if goLogFile != nil {
		log.Println("[Go] Closing log file")
		goLogFile.Close()
		goLogFile = nil
	}
}

func ReadGoLog() string {
	return ""
}

func New() *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	log.Println("[Engine] New engine instance created")
	return &Engine{
		ctx:          ctx,
		cancel:       cancel,
		status:       "disconnected",
		stats:        &engineStats{startTime: time.Now()},
		batteryLevel: 100,
		userSettings: &UserSettings{
			SocksPort:        7070,
			DialTimeout:      30,
			HandshakeTimeout: 15,
		},
	}
}

func (e *Engine) SetCallback(cb Callback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.callback = cb
	log.Println("[Engine] Callback set")
}

func (e *Engine) SetUserSettings(jsonStr string) bool {
	defer e.recoverPanic("SetUserSettings")

	log.Printf("[Engine] === SetUserSettings ===")
	log.Printf("[Engine] Input: %s", jsonStr)

	var settings UserSettings
	if err := json.Unmarshal([]byte(jsonStr), &settings); err != nil {
		log.Printf("[Engine] Parse error: %v", err)
		e.logError("Failed to parse user settings: " + err.Error())
		return false
	}

	e.mu.Lock()
	e.userSettings = &settings
	e.mu.Unlock()

	log.Printf("[Engine] Settings applied: socks=%d, dial=%ds, handshake=%ds", 
		settings.SocksPort, settings.DialTimeout, settings.HandshakeTimeout)

	e.logDebug(fmt.Sprintf("User settings updated: dial_timeout=%d, handshake_timeout=%d",
		settings.DialTimeout, settings.HandshakeTimeout))
	
	log.Println("[Engine] SetUserSettings SUCCESS ✅")
	return true
}

func (e *Engine) LoadConfigJSON(jsonStr string) bool {
	defer e.recoverPanic("LoadConfigJSON")

	log.Printf("[Engine] === LoadConfigJSON ===")
	log.Printf("[Engine] Config length: %d bytes", len(jsonStr))

	loader := config.NewLoader()
	cfg, err := loader.LoadFromJSON([]byte(jsonStr))
	if err != nil {
		log.Printf("[Engine] Config load failed: %v", err)
		e.logError("Config load failed: " + err.Error())
		return false
	}

	e.mu.Lock()
	e.config = cfg
	e.protocol = cfg.Server.Protocol
	e.mu.Unlock()

	log.Printf("[Engine] Config loaded: %s (%s)", cfg.Server.Name, cfg.Server.Protocol)
	e.logInfo(fmt.Sprintf("Config loaded: %s (%s)", cfg.Server.Name, cfg.Server.Protocol))
	return true
}

func (e *Engine) LoadConfigURI(uri string) bool {
	defer e.recoverPanic("LoadConfigURI")

	log.Printf("[Engine] === LoadConfigURI ===")
	log.Printf("[Engine] URI: %s", uri)

	loader := config.NewLoader()
	cfg, err := loader.LoadFromURI(uri)
	if err != nil {
		log.Printf("[Engine] URI load failed: %v", err)
		e.logError("URI load failed: " + err.Error())
		return false
	}

	e.mu.Lock()
	e.config = cfg
	e.protocol = cfg.Server.Protocol
	e.mu.Unlock()

	log.Printf("[Engine] Config loaded from URI: %s", cfg.Server.Name)
	e.logInfo(fmt.Sprintf("Config loaded from URI: %s", cfg.Server.Name))
	return true
}

func (e *Engine) LoadPreset(presetName string) bool {
	defer e.recoverPanic("LoadPreset")

	log.Printf("[Engine] === LoadPreset: %s ===", presetName)

	cfg, ok := config.GetPreset(presetName)
	if !ok {
		log.Printf("[Engine] Preset not found: %s", presetName)
		e.logError("Preset not found: " + presetName)
		return false
	}

	e.mu.Lock()
	e.config = cfg
	e.protocol = cfg.Server.Protocol
	e.mu.Unlock()

	log.Printf("[Engine] Preset loaded: %s", presetName)
	e.logInfo(fmt.Sprintf("Preset loaded: %s", presetName))
	return true
}

func (e *Engine) ExportConfigURI() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.config == nil {
		log.Println("[Engine] ExportConfigURI: no config")
		return ""
	}

	loader := config.NewLoader()
	uri, err := loader.ExportToURI(e.config)
	if err != nil {
		log.Printf("[Engine] ExportConfigURI failed: %v", err)
		return ""
	}

	log.Printf("[Engine] Config exported to URI (%d chars)", len(uri))
	return uri
}

func (e *Engine) ExportConfigJSON() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.config == nil {
		log.Println("[Engine] ExportConfigJSON: no config")
		return ""
	}

	loader := config.NewLoader()
	data, err := loader.ExportToJSON(e.config, true)
	if err != nil {
		log.Printf("[Engine] ExportConfigJSON failed: %v", err)
		return ""
	}

	log.Printf("[Engine] Config exported to JSON (%d bytes)", len(data))
	return string(data)
}

func (e *Engine) SetBatteryLevel(level int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.batteryLevel = level
	log.Printf("[Engine] Battery level: %d%%", level)
	e.logDebug(fmt.Sprintf("Battery level: %d%%", level))

	if e.adaptiveCover != nil {
		e.adaptiveCover.SetBatteryLevel(level)
		
		if e.config != nil && e.config.Cover.Adaptive.BatteryAware && level < 20 {
			log.Printf("[Engine] Low battery warning (%d%%)", level)
			e.logWarn(fmt.Sprintf("Low battery (%d%%) - reducing cover activity", level))
		}
	}
}

func (e *Engine) SetDataSaverMode(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.dataSaverMode = enabled
	log.Printf("[Engine] Data saver mode: %v", enabled)
	
	if e.config != nil {
		e.config.Cover.Adaptive.DataSaverMode = enabled
	}

	if e.adaptiveCover != nil {
		e.adaptiveCover.SetDataSaverMode(enabled)
	}

	e.logInfo(fmt.Sprintf("Data saver mode: %v", enabled))
}

func (e *Engine) Connect() bool {
	defer e.recoverPanic("Connect")

	log.Println("[Engine] === Connect ===")

	e.mu.Lock()
	if e.status == "connected" || e.status == "connecting" {
		log.Printf("[Engine] Already in state: %s", e.status)
		e.mu.Unlock()
		return false
	}

	if e.config == nil {
		e.mu.Unlock()
		log.Println("[Engine] ERROR: No config loaded")
		e.logError("No config loaded")
		return false
	}

	log.Printf("[Engine] Server: %s", e.config.Server.Address)
	log.Printf("[Engine] Protocol: %s", e.config.Server.Protocol)

	e.setStatus("connecting")
	e.retryCount = 0
	e.mu.Unlock()

	go e.connectWithRetry()
	return true
}

func (e *Engine) connectWithRetry() {
	defer e.recoverPanic("connectWithRetry")

	log.Printf("[Engine] Connection attempt loop started (max: %d)", MaxRetryAttempts)

	for e.retryCount < MaxRetryAttempts {
		log.Printf("[Engine] Attempt %d/%d...", e.retryCount+1, MaxRetryAttempts)
		e.logInfo(fmt.Sprintf("Connection attempt %d/%d...", e.retryCount+1, MaxRetryAttempts))

		err := e.connectInternal()
		if err == nil {
			e.setStatus("connected")
			log.Println("[Engine] ✓ Connected successfully!")
			e.logInfo("✓ Connected successfully!")
			return
		}

		log.Printf("[Engine] Attempt %d failed: %v", e.retryCount+1, err)
		e.logError(fmt.Sprintf("Attempt %d failed: %v", e.retryCount+1, err))
		e.retryCount++

		if e.retryCount < MaxRetryAttempts {
			log.Printf("[Engine] Retrying in %v...", RetryDelay)
			e.logInfo(fmt.Sprintf("Retrying in %v...", RetryDelay))
			time.Sleep(RetryDelay)
		}
	}

	log.Println("[Engine] All connection attempts failed")

	if e.config.DNS.Enabled && e.config.DNS.AutoSwitch {
		log.Println("[Engine] Attempting DNS fallback...")
		e.logWarn("All TLS attempts failed - trying DNS fallback...")
		if err := e.enableDNSFallback(); err != nil {
			log.Printf("[Engine] DNS fallback also failed: %v", err)
			e.logError("DNS fallback also failed: " + err.Error())
			e.setStatus("disconnected")
		} else {
			e.setStatus("connected")
			log.Println("[Engine] ✓ Connected via DNS fallback!")
			e.logInfo("✓ Connected via DNS fallback!")
		}
	} else {
		e.setStatus("disconnected")
	}
}

func (e *Engine) connectInternal() error {
	log.Println("[Engine] === connectInternal ===")
	
	e.mu.RLock()
	cfg := e.config
	protocol := e.protocol
	e.mu.RUnlock()

	log.Printf("[Engine] Protocol: %s", protocol)
	log.Printf("[Engine] Cover enabled: %v", cfg.Cover.Enabled)
	log.Printf("[Engine] SNI enabled: %v", cfg.SNI.Enabled)
	log.Printf("[Engine] DNS fallback enabled: %v", cfg.DNS.Enabled)

	if cfg.SNI.Enabled {
		log.Println("[Engine] Initializing SNI manager...")
		sniMgr, err := sni.NewManagerFromConfig(&cfg.SNI)
		if err != nil {
			log.Printf("[Engine] SNI manager init failed: %v", err)
			e.logWarn("SNI manager init failed: " + err.Error())
		} else {
			e.mu.Lock()
			e.sniManager = sniMgr
			e.mu.Unlock()

			currentSNI := sniMgr.Get()
			e.stats.mu.Lock()
			e.stats.currentSNI = currentSNI
			e.stats.mu.Unlock()

			log.Printf("[Engine] SNI rotation enabled (%s mode, current: %s)", cfg.SNI.Mode, currentSNI)
			e.logInfo(fmt.Sprintf("SNI rotation enabled (%s mode, current: %s)", cfg.SNI.Mode, currentSNI))
		}
	}

	var coverMgr *cover.Manager
	if cfg.Cover.Enabled {
		log.Println("[Engine] Initializing cover traffic...")
		coverCfg := e.buildCoverConfig(&cfg.Cover)
		
		maxPadding := config.GetMaxPaddingForMode(cfg.Cover.Mode)
		modeCfg := &cover.ModeConfig{
			MaxPadding: maxPadding,
		}
		
		adaptiveCover := cover.NewAdaptiveCover(modeCfg)

		e.mu.RLock()
		currentBattery := e.batteryLevel
		currentDataSaver := e.dataSaverMode
		e.mu.RUnlock()
		
		adaptiveCover.SetBatteryLevel(currentBattery)
		log.Printf("[Engine] Initial battery level: %d%%", currentBattery)

		if cfg.Cover.Adaptive.BatteryAware {
			adaptiveCover.SetBatteryAware(true)
			if currentBattery < 20 {
				log.Println("[Engine] Low battery detected - reducing cover activity")
			}
		}
		
		if cfg.Cover.Adaptive.DataSaverMode || currentDataSaver {
			adaptiveCover.SetDataSaverMode(true)
			log.Println("[Engine] Data saver mode enabled")
		}
		
		coverMgr = cover.NewManager(coverCfg, adaptiveCover)
		
		e.mu.Lock()
		e.coverManager = coverMgr
		e.adaptiveCover = adaptiveCover
		e.mu.Unlock()

		coverMgr.Start(e.ctx)
		log.Printf("[Engine] Cover traffic enabled (%s mode, %d domains)", cfg.Cover.Mode, len(cfg.Cover.Domains))

		e.logInfo("[Engine] warming up (waiting 3 seconds for initial requests)...")
		log.Println("[Engine] Cover warm-up: 3 seconds...")
		
		warmupTimer := time.NewTimer(3 * time.Second)
		select {
		case <-warmupTimer.C:
			log.Println("[Engine] Cover warm-up complete")
			e.logInfo("[Engine] warm-up complete, ready to connect")
		case <-e.ctx.Done():
			warmupTimer.Stop()
			return e.ctx.Err()
		}
	}

	switch strings.ToLower(protocol) {
	case "grouk":
		log.Println("[Engine] Using Grouk protocol (UDP)")
		return e.connectGrouk(cfg, coverMgr)
	case "zhip":
		log.Println("[Engine] Using Zhip protocol (QUIC)")
		return e.connectZhip(cfg, coverMgr)
	default:
		log.Println("[Engine] Using Guarch protocol (TLS)")
		return e.connectGuarch(cfg, coverMgr)
	}
}

func (e *Engine) connectGuarch(cfg *config.ServerConfig, coverMgr *cover.Manager) error {
	log.Println("[Engine] === connectGuarch ===")

	e.mu.RLock()
	userSettings := e.userSettings
	e.mu.RUnlock()

	configWithTimeouts := e.mergeTimeoutSettings(cfg, userSettings)

	log.Printf("[Engine] Creating connector for %s...", cfg.Server.Address)
	connector := engine.NewConnector(configWithTimeouts)

	if e.sniManager != nil {
		currentSNI := e.sniManager.Get()
		connector.SetSNI(currentSNI)
		log.Printf("[Engine] Using SNI: %s", currentSNI)
		e.logInfo(fmt.Sprintf("Using SNI: %s", currentSNI))
	}

	if coverMgr != nil {
		log.Println("[Engine] Sending initial cover request...")
		coverMgr.SendOne()
	}

	log.Println("[Engine] Dialing server...")
	rawConn, err := connector.Dial(e.ctx)
	if err != nil {
		log.Printf("[Engine] Dial failed: %v", err)
		return fmt.Errorf("connector dial failed: %w", err)
	}
	log.Println("[Engine] TCP connection established ✅")

	maxPadding := config.GetMaxPaddingForMode(cfg.Cover.Mode)
	
	handshakeCfg := &transport.HandshakeConfig{
		PSK:            []byte(cfg.Server.PSK),
		MaxPadding:     maxPadding,
		PaddingEnabled: maxPadding > 0,
	}

	log.Printf("[Engine] Performing handshake (padding: %d)...", maxPadding)
	sc, err := transport.Handshake(rawConn, false, handshakeCfg)
	if err != nil {
		rawConn.Close()
		log.Printf("[Engine] Handshake failed: %v", err)
		return fmt.Errorf("handshake failed: %w", err)
	}
	log.Println("[Engine] Handshake complete ✅")

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	log.Println("[Engine] Creating multiplexer...")
	m := mux.NewMux(sc, true)

	e.mu.Lock()
	e.stats.connectTime = time.Now()
	e.connector = connector
	e.mu.Unlock()

	log.Println("[Engine] Starting SOCKS5 server...")
	return e.startSOCKS5(func() (io.ReadWriteCloser, error) {
		return m.OpenStream()
	})
}

func (e *Engine) mergeTimeoutSettings(cfg *config.ServerConfig, userSettings *UserSettings) *config.ServerConfig {
	merged := *cfg

	if merged.Transport == nil {
		merged.Transport = &config.TransportConfig{}
	}

	if merged.Transport.DialTimeout == 0 && userSettings.DialTimeout > 0 {
		merged.Transport.DialTimeout = userSettings.DialTimeout
		log.Printf("[Engine] Using user dial_timeout: %ds", userSettings.DialTimeout)
		e.logDebug(fmt.Sprintf("Using user dial_timeout: %ds", userSettings.DialTimeout))
	} else if merged.Transport.DialTimeout > 0 {
		log.Printf("[Engine] Using config dial_timeout: %ds", merged.Transport.DialTimeout)
		e.logDebug(fmt.Sprintf("Using config dial_timeout: %ds", merged.Transport.DialTimeout))
	}

	if merged.Transport.HandshakeTimeout == 0 && userSettings.HandshakeTimeout > 0 {
		merged.Transport.HandshakeTimeout = userSettings.HandshakeTimeout
		log.Printf("[Engine] Using user handshake_timeout: %ds", userSettings.HandshakeTimeout)
		e.logDebug(fmt.Sprintf("Using user handshake_timeout: %ds", userSettings.HandshakeTimeout))
	} else if merged.Transport.HandshakeTimeout > 0 {
		log.Printf("[Engine] Using config handshake_timeout: %ds", merged.Transport.HandshakeTimeout)
		e.logDebug(fmt.Sprintf("Using config handshake_timeout: %ds", merged.Transport.HandshakeTimeout))
	}

	return &merged
}

func (e *Engine) connectGrouk(cfg *config.ServerConfig, coverMgr *cover.Manager) error {
	log.Println("[Engine] === connectGrouk ===")

	serverAddr := cfg.Server.Address
	log.Printf("[Engine] Resolving UDP address: %s", serverAddr)
	
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		log.Printf("[Engine] Resolve failed: %v", err)
		return fmt.Errorf("resolve failed: %w", err)
	}

	log.Println("[Engine] Creating UDP socket...")
	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Printf("[Engine] UDP listen failed: %v", err)
		return fmt.Errorf("UDP listen failed: %w", err)
	}

	udpConn.SetReadBuffer(4 * 1024 * 1024)
	udpConn.SetWriteBuffer(4 * 1024 * 1024)
	log.Println("[Engine] UDP buffers set (4MB each)")

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	log.Println("[Engine] Performing Grouk handshake...")
	session, err := transport.GroukClientHandshake(udpConn, udpAddr, []byte(cfg.Server.PSK))
	if err != nil {
		udpConn.Close()
		log.Printf("[Engine] Grouk handshake failed: %v", err)
		return fmt.Errorf("Grouk handshake failed: %w", err)
	}
	log.Println("[Engine] Grouk handshake complete ✅")

	e.mu.Lock()
	e.groukSession = session
	e.groukUDP = udpConn
	e.stats.connectTime = time.Now()
	e.mu.Unlock()

	log.Println("[Engine] Starting Grouk read loop...")
	go e.groukReadLoop(session, udpConn, udpAddr)

	return e.startSOCKS5(func() (io.ReadWriteCloser, error) {
		return session.OpenStream()
	})
}

func (e *Engine) groukReadLoop(session *transport.GroukSession, udpConn *net.UDPConn, serverAddr *net.UDPAddr) {
	defer e.recoverPanic("groukReadLoop")

	log.Println("[Engine] Grouk read loop started")
	buf := make([]byte, 2048)
	for {
		select {
		case <-e.ctx.Done():
			log.Println("[Engine] Grouk read loop stopped")
			return
		default:
		}

		udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		if !addr.IP.Equal(serverAddr.IP) || addr.Port != serverAddr.Port {
			continue
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		pkt, err := transport.UnmarshalGroukPacket(data)
		if err != nil {
			continue
		}

		if pkt.SessionID == session.ID {
			session.HandlePacketFromClient(pkt)
		}
	}
}

func (e *Engine) connectZhip(cfg *config.ServerConfig, coverMgr *cover.Manager) error {
	log.Println("[Engine] === connectZhip ===")

	serverAddr := cfg.Server.Address
	log.Printf("[Engine] Dialing QUIC server: %s", serverAddr)

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	conn, err := transport.ZhipDial(e.ctx, serverAddr, cfg.Server.CertPin, nil)
	if err != nil {
		log.Printf("[Engine] QUIC dial failed: %v", err)
		return fmt.Errorf("QUIC dial failed: %w", err)
	}
	log.Println("[Engine] QUIC connection established ✅")

	log.Println("[Engine] Performing Zhip authentication...")
	if err := transport.ZhipClientAuth(conn, []byte(cfg.Server.PSK)); err != nil {
		conn.CloseWithError(0, "auth failed")
		log.Printf("[Engine] Zhip auth failed: %v", err)
		return fmt.Errorf("Zhip auth failed: %w", err)
	}
	log.Println("[Engine] Zhip auth complete ✅")

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	e.mu.Lock()
	e.stats.connectTime = time.Now()
	e.mu.Unlock()

	return e.startSOCKS5(func() (io.ReadWriteCloser, error) {
		return conn.OpenStreamSync(e.ctx)
	})
}

func (e *Engine) enableDNSFallback() error {
	log.Println("[Engine] === enableDNSFallback ===")

	e.mu.RLock()
	dnsCfg := &e.config.DNS
	e.mu.RUnlock()

	if !dnsCfg.Enabled {
		return fmt.Errorf("DNS fallback not enabled in config")
	}

	log.Printf("[Engine] Creating DNS client (domain: %s)...", dnsCfg.Domain)
	clientCfg := &dns.ClientConfig{
		Domain:     dnsCfg.Domain,
		DNSServers: dnsCfg.Servers,
		Timeout:    dnsCfg.Timeout.Duration,
		Retries:    dnsCfg.SwitchThreshold,
		RetryDelay: 500 * time.Millisecond,
	}
	
	dnsClient, err := dns.NewClient(clientCfg)
	if err != nil {
		log.Printf("[Engine] DNS client creation failed: %v", err)
		return fmt.Errorf("DNS client creation failed: %w", err)
	}

	e.mu.Lock()
	e.dnsClient = dnsClient
	e.mu.Unlock()

	e.usingDNSFallback.Store(true)

	if e.callback != nil {
		e.callback.OnDNSFallback(true)
	}

	log.Println("[Engine] DNS Fallback Mode Active ⚠️")
	e.logWarn("⚠️ DNS Fallback Mode Active (Reduced Speed ~50Kbps)")

	return e.startSOCKS5(func() (io.ReadWriteCloser, error) {
		sessionID := uint32(time.Now().UnixNano() & 0xFFFFFFFF)
		
		streamCfg := &dns.StreamConfig{
			RecvBufferSize: 65536,
			SendBufferSize: 32768,
			ReadTimeout:    0,
			WriteTimeout:   0,
			IdleTimeout:    5 * time.Minute,
			MaxRetries:     e.config.DNS.MaxRetries,
			RetryDelay:     e.config.DNS.RetryDelay.Duration,
			MaxPacketSize:  32768,
			Compression:    e.config.DNS.Compression,
		}
		
		if e.config.DNS.BufferSize > 0 {
			streamCfg.RecvBufferSize = e.config.DNS.BufferSize
			streamCfg.SendBufferSize = e.config.DNS.BufferSize / 2
		}
		
		if e.config.DNS.MaxPacketSize > 0 {
			streamCfg.MaxPacketSize = e.config.DNS.MaxPacketSize
		}
		
		wrapper := dns.NewStreamWrapperWithConfig(e.dnsClient, sessionID, streamCfg)
		
		log.Printf("[Engine] DNS stream created (session: %08x, buffer: %d)", sessionID, streamCfg.RecvBufferSize)
		e.logDebug(fmt.Sprintf("DNS stream created (session: %08x, buffer: %d)", sessionID, streamCfg.RecvBufferSize))
		
		return wrapper, nil
	})
}

func (e *Engine) startSOCKS5(openStream func() (io.ReadWriteCloser, error)) error {
	log.Println("[Engine] === startSOCKS5 ===")

	e.mu.RLock()
	socksPort := e.userSettings.SocksPort
	if socksPort == 0 {
		socksPort = 7070
	}
	e.mu.RUnlock()

	listenAddr := fmt.Sprintf("127.0.0.1:%d", socksPort)
	log.Printf("[Engine] Listening on %s...", listenAddr)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("[Engine] SOCKS5 listen failed: %v", err)
		return fmt.Errorf("SOCKS5 listen failed: %w", err)
	}

	e.mu.Lock()
	e.listener = ln
	e.mu.Unlock()

	log.Printf("[Engine] SOCKS5 server listening on %s ✅", listenAddr)
	e.logInfo(fmt.Sprintf("SOCKS5 server listening on %s", listenAddr))

	go e.statsReporter()
	go e.sniRotator()

	go func() {
		defer e.recoverPanic("SOCKS5 accept loop")

		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-e.ctx.Done():
					log.Println("[Engine] SOCKS5 accept loop stopped")
					return
				default:
					continue
				}
			}

			atomic.AddInt64(&e.stats.totalConnections, 1)
			go e.handleSOCKS5(conn, openStream)
		}
	}()

	return nil
}

func (e *Engine) handleSOCKS5(socksConn net.Conn, openStream func() (io.ReadWriteCloser, error)) {
	defer e.recoverPanic("handleSOCKS5")
	defer socksConn.Close()

	atomic.AddInt32(&e.stats.activeStreams, 1)
	defer atomic.AddInt32(&e.stats.activeStreams, -1)

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		return
	}

	stream, err := openStream()
	if err != nil {
		socks5.SendReply(socksConn, 0x01)
		return
	}
	defer stream.Close()

	if err := e.sendConnectRequest(stream, target); err != nil {
		socks5.SendReply(socksConn, 0x01)
		return
	}

	socks5.SendReply(socksConn, 0x00)
	e.relayWithStats(stream, socksConn)
}

func (e *Engine) sendConnectRequest(stream io.ReadWriter, target string) error {
	host, portStr, _ := net.SplitHostPort(target)
	port := parsePort(portStr)

	addrType := protocol.AddrTypeDomain
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			addrType = protocol.AddrTypeIPv4
		} else {
			addrType = protocol.AddrTypeIPv6
		}
	}

	req := &protocol.ConnectRequest{
		AddrType: addrType,
		Addr:     host,
		Port:     port,
	}

	reqData, err := req.Marshal()
	if err != nil {
		return err
	}

	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

	if _, err := stream.Write(lenBuf); err != nil {
		return err
	}
	if _, err := stream.Write(reqData); err != nil {
		return err
	}

	statusBuf := make([]byte, 1)
	if _, err := io.ReadFull(stream, statusBuf); err != nil {
		return err
	}

	if statusBuf[0] != protocol.ConnectSuccess {
		return fmt.Errorf("connect rejected")
	}

	return nil
}

func (e *Engine) relayWithStats(stream io.ReadWriteCloser, conn net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		defer e.recoverPanic("relay upload")
		buf := make([]byte, 32768)
		
		for {
			n, err := conn.Read(buf)
			
			if n > 0 {
				written, writeErr := stream.Write(buf[:n])
				
				if writeErr != nil {
					break 
				}
				
				if written > 0 {
					e.stats.mu.Lock()
					e.stats.totalUpload += int64(written)
					e.stats.mu.Unlock()
					if e.adaptiveCover != nil {
						e.adaptiveCover.RecordTraffic(int64(written))
					}
				}
			}
			
			if err != nil {
				break
			}
		}
		
		done <- struct{}{}
	}()

	go func() {
		defer e.recoverPanic("relay download")
		buf := make([]byte, 32768)
		
		for {
			n, err := stream.Read(buf)
			
			if n > 0 {
				written, writeErr := conn.Write(buf[:n])
				
				if writeErr != nil {
					break
				}
				
				if written > 0 {
					e.stats.mu.Lock()
					e.stats.totalDownload += int64(written)
					e.stats.mu.Unlock()
					
					if e.adaptiveCover != nil {
						e.adaptiveCover.RecordTraffic(int64(written))
					}
				}
			}
			
			if err != nil {
				break
			}
		}
		
		done <- struct{}{}
	}()

	<-done
	stream.Close()
	conn.Close()
	<-done
}

func (e *Engine) statsReporter() {
	defer e.recoverPanic("statsReporter")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastUp, lastDown int64

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.stats.mu.Lock()

			upSpeed := e.stats.totalUpload - lastUp
			downSpeed := e.stats.totalDownload - lastDown
			lastUp = e.stats.totalUpload
			lastDown = e.stats.totalDownload

			e.stats.lastSpeedUp = upSpeed
			e.stats.lastSpeedDown = downSpeed

			if e.adaptiveCover != nil {
				bytesPerMin := (upSpeed + downSpeed) * 60
				e.adaptiveCover.RecordTraffic(bytesPerMin)
				e.stats.activityLevel = e.adaptiveCover.GetCurrentLevel().String()
			}

			data := map[string]interface{}{
				"upload_speed":      upSpeed,
				"download_speed":    downSpeed,
				"total_upload":      e.stats.totalUpload,
				"total_download":    e.stats.totalDownload,
				"active_streams":    atomic.LoadInt32(&e.stats.activeStreams),
				"total_connections": e.stats.totalConnections,
				"duration_seconds":  int(time.Since(e.stats.startTime).Seconds()),
				"current_sni":       e.stats.currentSNI,
				"sni_switches":      e.stats.sniSwitches,
				"activity_level":    e.stats.activityLevel,
				"dns_fallback":      e.usingDNSFallback.Load(),
			}

			e.stats.mu.Unlock()

			jsonData, _ := json.Marshal(data)
			if e.callback != nil {
				e.callback.OnStatsUpdate(string(jsonData))
			}
		}
	}
}

func (e *Engine) sniRotator() {
	defer e.recoverPanic("sniRotator")

	if e.sniManager == nil {
		return
	}

	e.mu.RLock()
	var interval time.Duration
	if e.config.SNI.RotationInterval.Duration > 0 {
		interval = e.config.SNI.RotationInterval.Duration
	} else {
		interval = 5 * time.Minute
	}
	e.mu.RUnlock()

	log.Printf("[Engine] SNI rotator started (interval: %v)", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			log.Println("[Engine] SNI rotator stopped")
			return
		case <-ticker.C:
			newSNI := e.sniManager.Get()

			e.stats.mu.Lock()
			if newSNI != e.stats.currentSNI {
				oldSNI := e.stats.currentSNI
				e.stats.currentSNI = newSNI
				e.stats.sniSwitches++
				e.stats.mu.Unlock()

				log.Printf("[Engine] SNI rotated: %s → %s", oldSNI, newSNI)
				e.logInfo(fmt.Sprintf("SNI rotated: %s → %s", oldSNI, newSNI))

				if e.callback != nil {
					e.callback.OnSNIRotation(newSNI)
				}
			} else {
				e.stats.mu.Unlock()
			}
		}
	}
}

func (e *Engine) Disconnect() bool {
	defer e.recoverPanic("Disconnect")

	log.Println("[Engine] === Disconnect ===")

	e.mu.Lock()
	defer e.mu.Unlock()

	e.setStatus("disconnecting")

	e.StopTun()

	if e.cancel != nil {
		e.cancel()
	}

	if e.listener != nil {
		e.listener.Close()
		e.listener = nil
		log.Println("[Engine] SOCKS5 listener closed")
	}

	if e.muxConn != nil {
		e.muxConn.Close()
		e.muxConn = nil
		log.Println("[Engine] Mux closed")
	}

	if e.groukSession != nil {
		e.groukSession.Close()
		e.groukSession = nil
		log.Println("[Engine] Grouk session closed")
	}

	if e.groukUDP != nil {
		e.groukUDP.Close()
		e.groukUDP = nil
		log.Println("[Engine] Grouk UDP closed")
	}

	if e.sniManager != nil {
		e.sniManager.Stop()
		e.sniManager = nil
		log.Println("[Engine] SNI manager stopped")
	}

	if e.coverManager != nil {
		e.coverManager.Stop()
		e.coverManager = nil
		log.Println("[Engine] Cover manager stopped")
	}

	if e.dnsClient != nil {
		e.dnsClient.Close()
		e.dnsClient = nil
		log.Println("[Engine] DNS client closed")
	}

	e.usingDNSFallback.Store(false)
	e.setStatus("disconnected")
	log.Println("[Engine] ✓ Disconnected")
	e.logInfo("✓ Disconnected")

	return true
}

func (e *Engine) GetStatus() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

func (e *Engine) GetStats() string {
	e.stats.mu.RLock()
	defer e.stats.mu.RUnlock()

	data := map[string]interface{}{
		"total_upload":      e.stats.totalUpload,
		"total_download":    e.stats.totalDownload,
		"upload_speed":      e.stats.lastSpeedUp,
		"download_speed":    e.stats.lastSpeedDown,
		"active_streams":    atomic.LoadInt32(&e.stats.activeStreams),
		"total_connections": e.stats.totalConnections,
		"duration_seconds":  int(time.Since(e.stats.startTime).Seconds()),
		"current_sni":       e.stats.currentSNI,
		"sni_switches":      e.stats.sniSwitches,
		"activity_level":    e.stats.activityLevel,
		"dns_fallback":      e.usingDNSFallback.Load(),
	}

	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func (e *Engine) IsConnected() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status == "connected"
}

func GetVersion() string {
	return fmt.Sprintf("Guarch Mobile Engine v%s", Version)
}

func (e *Engine) buildCoverConfig(cfg *config.CoverConfig) *cover.Config {
	domains := make([]cover.DomainConfig, len(cfg.Domains))
	for i, d := range cfg.Domains {
		domains[i] = cover.DomainConfig{
			Domain:      d.Domain,
			Paths:       d.Paths,
			Weight:      d.Weight,
			MinInterval: d.IntervalMin.Duration,
			MaxInterval: d.IntervalMax.Duration,
		}
	}
	
	return &cover.Config{
		Enabled:       cfg.Enabled,
		Domains:       domains,
		MaxConcurrent: 3,
		IdleTraffic:   true,
	}
}

func (e *Engine) setStatus(s string) {
	e.status = s
	if e.callback != nil {
		e.callback.OnStatusChanged(s)
	}
}

func (e *Engine) logDebug(msg string) {
	log.Println("[DEBUG]", msg)
	if e.callback != nil {
		e.callback.OnLog("debug", msg)
	}
}

func (e *Engine) logInfo(msg string) {
	log.Println("[INFO]", msg)
	if e.callback != nil {
		e.callback.OnLog("info", msg)
	}
}

func (e *Engine) logWarn(msg string) {
	log.Println("[WARN]", msg)
	if e.callback != nil {
		e.callback.OnLog("warn", msg)
	}
}

func (e *Engine) logError(msg string) {
	log.Println("[ERROR]", msg)
	if e.callback != nil {
		e.callback.OnLog("error", msg)
		e.callback.OnError(msg)
	}
}

func (e *Engine) recoverPanic(funcName string) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("PANIC in %s: %v\n%s", funcName, r, debug.Stack())
		log.Println("[PANIC]", msg)
		e.logError(msg)
		e.setStatus("disconnected")
	}
}

func parsePort(s string) uint16 {
	var p uint16
	for _, c := range s {
		if c >= '0' && c <= '9' {
			p = p*10 + uint16(c-'0')
		}
	}
	return p
}
