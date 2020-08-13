// Diode Network Client
// Copyright 2019 IoT Blockchain Technology Corporation LLC (IBTC)
// Licensed under the Diode License, Version 1.0
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	// "net"
	// "regexp"
	// "strconv"
	// "strings"
	// "gopkg.in/yaml.v2"

	"github.com/diodechain/diode_go_client/pkg/diode/config"
	"github.com/diodechain/diode_go_client/pkg/diode/crypto"
	"github.com/diodechain/diode_go_client/pkg/diode/db"
	"github.com/diodechain/diode_go_client/pkg/diode/edge"
	"github.com/diodechain/diode_go_client/pkg/diode/rpc"
	"github.com/diodechain/diode_go_client/pkg/diode/util"
	"github.com/diodechain/log15"
	log "github.com/diodechain/log15"
	"github.com/spf13/cobra"
)

const (
	PublicPublishedMode = 1 << iota
	ProtectedPublishedMode
	PrivatePublishedMode
	LogToConsole = 1 << iota
	LogToFile
	TCPProtocol = 1 << iota
	UDPProtocol
	TLSProtocol
	AnyProtocol
	AverageBlockTime = 15
)

var (
	// === copy from flag
	AppConfig *config.Config
	finalText = `
Run 'diode COMMAND --help' for more information on a command.
`
	bootDiodeAddrs = []string{
		"asia.testnet.diode.io:41046",
		"europe.testnet.diode.io:41046",
		"usa.testnet.diode.io:41046",
	}
	NullAddr                   = [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	DefaultRegistryAddr        = [20]byte{80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	DefaultFleetAddr           = [20]byte{96, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	errWrongDiodeAddrs         = fmt.Errorf("wrong remote diode addresses")
	errConfigNotLoadedFromFile = fmt.Errorf("config wasn't loaded from file")
	// ======
	version     string = "development"
	buildTime   string
	socksServer *rpc.Server
	proxyServer *rpc.ProxyServer
	// configAPIServer *ConfigAPIServer
	pool       *rpc.DataPool
	bnsPattern = regexp.MustCompile(`^[0-9A-Za-z-]+$`)
	// ======
	// === copy from flag
	diodeCmd = &cobra.Command{
		Use:               "diode",
		Short:             "Diode network command line interface",
		Long:              "Hello, Diode! TODO: update this text!",
		PersistentPreRun:  prepareDiodeApp,
		PersistentPostRun: closeDiodeApp,
	}
	app Diode
	// main client, should remove
	client  *rpc.RPCClient
	clients map[util.Address]*rpc.RPCClient
)

func init() {
	// setup flag
	cobra.OnInitialize(initConfig)
	var cfg = &config.Config{}
	diodeCmd.PersistentFlags().StringVar(&cfg.DBPath, "dbpath", util.DefaultDBPath(), "file path to db file")
	diodeCmd.PersistentFlags().IntVar(&cfg.RetryTimes, "retrytimes", 3, "retry times to connect the remote rpc server")
	diodeCmd.PersistentFlags().BoolVar(&cfg.EnableEdgeE2E, "e2e", false, "enable edge e2e when start diode")
	// should put to httpd or other command
	// diodeCmd.PersistentFlags().BoolVar(&cfg.EnableUpdate, "update", false, "enable update when start diode")
	diodeCmd.PersistentFlags().BoolVar(&cfg.EnableMetrics, "metrics", false, "enable metrics stats")
	diodeCmd.PersistentFlags().BoolVar(&cfg.Debug, "debug", false, "turn on debug mode")
	diodeCmd.PersistentFlags().BoolVar(&cfg.EnableAPIServer, "api", false, "turn on the config api")
	diodeCmd.PersistentFlags().StringVar(&cfg.APIServerAddr, "apiaddr", "localhost:1081", "define config api server address")
	diodeCmd.PersistentFlags().IntVar(&cfg.RlimitNofile, "rlimit_nofile", 0, "specify the file descriptor numbers that can be opened by this process")
	diodeCmd.PersistentFlags().StringVar(&cfg.LogFilePath, "logfilepath", "", "file path to log file")
	diodeCmd.PersistentFlags().BoolVar(&cfg.LogDateTime, "logdatetime", false, "show the date time in log")
	diodeCmd.PersistentFlags().StringVar(&cfg.ConfigFilePath, "configpath", "", "yaml file path to config file")
	diodeCmd.PersistentFlags().StringVar(&cfg.CPUProfile, "cpuprofile", "", "file path for cpu profiling")
	diodeCmd.PersistentFlags().StringVar(&cfg.MEMProfile, "memprofile", "", "file path for memory profiling")

	var fleetFake string
	diodeCmd.PersistentFlags().StringVar(&fleetFake, "fleet", "", "setup fleet address for diode client")
	diodeCmd.PersistentFlags().MarkDeprecated("fleet", "@deprecated. Use: 'diode config set fleet=0x1234' instead")

	// tcp keepalive for node connection
	diodeCmd.PersistentFlags().BoolVar(&cfg.EnableKeepAlive, "keepalive", runtime.GOOS != "windows", "enable tcp keepalive (only Linux >= 2.4, DragonFly, FreeBSD, NetBSD and OS X >= 10.8 are supported)")
	diodeCmd.PersistentFlags().IntVar(&cfg.KeepAliveCount, "keepalivecount", 4, "the maximum number of keepalive probes TCP should send before dropping the connection")
	// keepaliveIdle := diodeCmd.PersistentFlags().Int("keepaliveidle", 30, "the time (in seconds) the connection needs to remain idle before TCP starts sending keepalive probes")
	// keepaliveInterval := diodeCmd.PersistentFlags().Int("keepaliveinterval", 5, "the time (in seconds) between individual keepalive probes")

	// remoteRPCTimeout := diodeCmd.PersistentFlags().Int("timeout", 5, "timeout seconds to connect to the remote rpc server")
	// retryWait := diodeCmd.PersistentFlags().Int("retrywait", 1, "wait seconds before next retry")
	diodeCmd.PersistentFlags().StringSliceVar(&cfg.RemoteRPCAddrs, "diodeaddrs", bootDiodeAddrs, "addresses of Diode node server (default: asia.testnet.diode.io:41046, europe.testnet.diode.io:41046, usa.testnet.diode.io:41046)")
	// diodeCmd.PersistentFlags().StringSliceVar(&cfg.SBlocklists, "blocklists", "addresses are not allowed to connect to published resource (worked when allowlists is empty)")
	// diodeCmd.PersistentFlags().StringSliceVar(&cfg.SAllowlists, "allowlists", "addresses are allowed to connect to published resource (worked when blocklists is empty)")
	// diodeCmd.PersistentFlags().StringSliceVar(&cfg.SBinds, "bind", "bind a remote port to a local port. -bind <local_port>:<to_address>:<to_port>:(udp|tcp)")
	if len(cfg.LogFilePath) > 0 {
		// TODO: logrotate?
		cfg.LogMode = LogToFile
	} else {
		cfg.LogMode = LogToConsole
	}
	cfg.Logger = newLogger(cfg)
	AppConfig = cfg
	// add subcommands
	diodeCmd.AddCommand(configCmd)
	diodeCmd.AddCommand(updateCmd)
	diodeCmd.AddCommand(timeCmd)
	diodeCmd.AddCommand(resetCmd)
	diodeCmd.AddCommand(bnsCmd)
}

func prepareDiodeApp(cmd *cobra.Command, args []string) {
	app = NewDiode(AppConfig, pool, clients)
	err := app.Init()
	if err != nil {
		return
	}
	return
}

func closeDiodeApp(cmd *cobra.Command, args []string) {
	if app.Started() {
		app.Close()
	}
}

func newLogger(cfg *config.Config) log.Logger {
	var logHandler log.Handler
	logger := log.New()
	if (cfg.LogMode & LogToConsole) > 0 {
		logHandler = log.StreamHandler(os.Stderr, log.TerminalFormat(cfg.LogDateTime))
	} else if (cfg.LogMode & LogToFile) > 0 {
		var err error
		logHandler, err = log.FileHandler(cfg.LogFilePath, log.TerminalFormat(cfg.LogDateTime))
		if err != nil {
			// panicWithError(err)
		}
	}
	logger.SetHandler(logHandler)
	return logger
}

// initConfig load file config
func initConfig() {}

func printLabel(label string, value string) {
	msg := fmt.Sprintf("%-20s : %-80s", label, value)
	AppConfig.Logger.Info(msg)
}

func printError(msg string, err error) {
	AppConfig.Logger.Error(msg, "error", err)
}

func printInfo(msg string) {
	AppConfig.Logger.Info(msg)
}

func connect(c chan *rpc.RPCClient, host string, cfg *config.Config, wg *sync.WaitGroup, pool *rpc.DataPool) {
	client, err := rpc.DoConnect(host, cfg, pool)
	if err != nil {
		cfg.Logger.Error(fmt.Sprintf("Connection to host: %s failed: %+v", host, err))
		wg.Done()
	} else {
		c <- client
	}
}

// ensure account state has been changed
// since account state will change after transaction
// we try to confirm the transactions by validate the account state
// to prevent from fork, maybe wait more blocks
func watchAccount(client *rpc.RPCClient, to util.Address) (res bool) {
	var bn uint64
	var startBN uint64
	var err error
	var oact *edge.Account
	var getTimes int
	var isConfirmed bool
	startBN, _ = client.LastValid()
	bn = startBN
	oact, _ = client.GetValidAccount(uint64(bn), to)
	for {
		<-time.After(15 * time.Second)
		var nbn uint64
		nbn, _ = client.LastValid()
		if nbn == bn {
			printInfo("Waiting for next valid block...")
			continue
		}
		var nact *edge.Account
		bn = nbn
		nact, err = client.GetValidAccount(uint64(bn), to)
		if err != nil {
			printInfo("Waiting for next valid block...")
			continue
		}
		if nact != nil {
			if oact == nil {
				isConfirmed = true
				break
			}
			if !bytes.Equal(nact.StateRoot(), oact.StateRoot()) {
				isConfirmed = true
				break
			}
			// state didn't change, maybe zero transaction, or block didn't include transaction?!
		}
		if getTimes == 15 || isConfirmed {
			break
		}
		getTimes++
	}
	return isConfirmed
}

// Diode represents didoe application
type Diode struct {
	config     *config.Config
	clientPool map[util.Address]*rpc.RPCClient
	datapool   *rpc.DataPool
	started    bool
	mx         sync.Mutex
}

// NewDiode return diode application
func NewDiode(cfg *config.Config, datapool *rpc.DataPool, pool map[util.Address]*rpc.RPCClient) Diode {
	return Diode{
		config:     cfg,
		clientPool: pool,
		datapool:   datapool,
	}
}

// Init initialize the diode application
func (dio *Diode) Init() error {
	// Connect to first server to respond, and keep the other connections opened
	cfg := dio.config
	pool = rpc.NewPool()

	printLabel("Diode Client version", fmt.Sprintf("%s %s", version, buildTime))

	// Initialize db
	clidb, err := db.OpenFile(cfg.DBPath)
	if err != nil {
		printError("Couldn't open database", err)
		return err
	}
	db.DB = clidb

	if version != "development" && cfg.EnableUpdate {
		var lastUpdateAtByt []byte
		var lastUpdateAt time.Time
		var shouldUpdateDiode bool
		lastUpdateAtByt, err = db.DB.Get("last_update_at")
		if err != nil {
			lastUpdateAt = time.Now()
			shouldUpdateDiode = true
		} else {
			lastUpdateAtInt := util.DecodeBytesToInt(lastUpdateAtByt)
			lastUpdateAt = time.Unix(int64(lastUpdateAtInt), 0)
			diff := time.Since(lastUpdateAt)
			shouldUpdateDiode = diff.Hours() >= 24
		}
		if shouldUpdateDiode {
			lastUpdateAt = time.Now()
			lastUpdateAtByt = util.DecodeInt64ToBytes(lastUpdateAt.Unix())
			db.DB.Put("last_update_at", lastUpdateAtByt)
			ret := doUpdate()
			if ret != 0 {
				return ErrFailedToUpdateClient

			}
		}
		return nil
	}

	if cfg.CPUProfile != "" {
		fd, err := os.Create(cfg.CPUProfile)
		if err != nil {
			printError("Couldn't open cpu profile file", err)
			return err
		}
		pprof.StartCPUProfile(fd)
		defer pprof.StopCPUProfile()
	}

	if cfg.MEMProfile != "" {
		mfd, err := os.Create(cfg.MEMProfile)
		if err != nil {
			printError("Couldn't open memory profile file", err)
			return err
		}
		runtime.GC()
		pprof.WriteHeapProfile(mfd)
		mfd.Close()
	}

	{
		if cfg.FleetAddr == config.NullAddr {
			cfg.FleetAddr = config.DefaultFleetAddr
		}

		cfg.ClientAddr = util.PubkeyToAddress(rpc.LoadClientPubKey())

		if !cfg.LoadFromFile {
			fleetAddr, err := db.DB.Get("fleet")
			if err != nil {
				// Migration if existing
				fleetAddr, err = db.DB.Get("fleet_id")
				if err == nil {
					cfg.FleetAddr, err = util.DecodeAddress(string(fleetAddr))
					if err == nil {
						db.DB.Put("fleet", cfg.FleetAddr[:])
						db.DB.Del("fleet_id")
					}
				}
			} else {
				copy(cfg.FleetAddr[:], fleetAddr)
			}
		}
	}
	printLabel("Client address", cfg.ClientAddr.HexString())
	printLabel("Fleet address", cfg.FleetAddr.HexString())
	return nil
}

// Start the diode application
func (dio *Diode) Start() error {
	cfg := dio.config
	wg := &sync.WaitGroup{}
	rpcAddrLen := len(cfg.RemoteRPCAddrs)
	if rpcAddrLen < 1 {
		return fmt.Errorf("should use at least one rpc address")
	}
	c := make(chan *rpc.RPCClient, rpcAddrLen)
	wg.Add(1)
	for _, RemoteRPCAddr := range cfg.RemoteRPCAddrs {
		go connect(c, RemoteRPCAddr, cfg, wg, pool)
	}
	var lvbn uint64
	var lvbh crypto.Sha3

	clientPool := make(map[util.Address]*rpc.RPCClient)
	go func() {
		for rpcClient := range c {
			// lvbn, lvbh = rpcClient.LastValid()
			// printLabel("Last valid block", fmt.Sprintf("%v %v", lvbn, util.EncodeToString(lvbh[:])))
			cfg.Logger.Info(fmt.Sprintf("Connected to host: %s, validating...", rpcClient.Host()))
			isValid, err := rpcClient.ValidateNetwork()
			if isValid {
				if client == nil {
					client = rpcClient
					wg.Done()
				}
				serverID, err := rpcClient.GetServerID()
				if err != nil {
					cfg.Logger.Warn("Failed to get server id: %v", err)
					rpcClient.Close()
					continue
				}
				clientPool[serverID] = rpcClient
			} else {
				if err != nil {
					cfg.Logger.Error(fmt.Sprintf("Network is not valid (err: %s), trying next...", err.Error()))
				} else {
					cfg.Logger.Error("Network is not valid for unknown reasons")
				}
				rpcClient.Close()
			}
		}
		// should end waiting if there is no valid client
		wg.Done()
	}()
	wg.Wait()
	dio.clientPool = clientPool

	if client == nil {
		err := fmt.Errorf("server are not validated")
		printError("Couldn't connect to any server", err)
		return err
	}
	lvbn, lvbh = client.LastValid()
	cfg.Logger.Info(fmt.Sprintf("Network is validated, last valid block: %d 0x%x", lvbn, lvbh))
	dio.started = true
	return nil
}

// Started returns the whether diode application has been started
func (dio *Diode) Started() bool {
	dio.mx.Lock()
	defer dio.mx.Unlock()
	return dio.started
}

// Close shut down diode application
func (dio *Diode) Close() {
	fmt.Println("1/7 Stopping client")
	// client.Close()
	for _, cc := range dio.clientPool {
		cc.Close()
	}
	fmt.Println("2/7 Stopping socksserver")
	// socksServer.Close()
	fmt.Println("3/7 Stopping proxyserver")
	// if proxyServer != nil {
	// 	proxyServer.Close()
	// }
	// fmt.Println("4/7 Stopping configserver")
	// if configAPIServer != nil {
	// 	configAPIServer.Close()
	// }
	fmt.Println("5/7 Cleaning pool")
	if pool != nil {
		pool.Close()
	}
	fmt.Println("6/7 Closing logs")
	handler := AppConfig.Logger.GetHandler()
	if closingHandler, ok := handler.(log15.ClosingHandler); ok {
		closingHandler.WriteCloser.Close()
	}
	fmt.Println("7/7 Exiting")
}

// Execute the diode command
func Execute() error {
	// keepaliveIntervalTime, err := time.ParseDuration(strconv.Itoa(*keepaliveInterval) + "s")
	// cfg.KeepAliveInterval = keepaliveIntervalTime
	// if err != nil {
	// 	return err
	// }
	// keepaliveIdleTime, err := time.ParseDuration(strconv.Itoa(*keepaliveIdle) + "s")
	// cfg.KeepAliveIdle = keepaliveIdleTime
	// if err != nil {
	// 	return err
	// }
	return diodeCmd.Execute()
}