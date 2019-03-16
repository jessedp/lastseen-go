package main

import (
	"flag"
	"io/ioutil"
	"os"
	"time"

	"encoding/json"
	"github.com/atrox/homedir"
	"github.com/godbus/dbus"
	"github.com/jessedp/lastseen-go/version"
	"github.com/sevlyar/go-daemon"
	"github.com/sirupsen/logrus"
	//"gopkg.in/natefinch/lumberjack.v2"
	"bytes"
	"fmt"
	"github.com/manifoldco/promptui"
	"net/http"
	"syscall"
)

const (
	//console output stuff

	// SEP is a separator
	SEP = "-------------------------------------------------------------------\n"

	// BANNER is what is printed for help/info output
	BANNER = `
 __     __   ____  ____  ____  ____  ____  __ _ 
(  )   / _\ / ___)(_  _)/ ___)(  __)(  __)(  ( \
/ (_/\/    \\___ \  )(  \___ \ ) _)  ) _) /    /
\____/\_/\_/(____/ (__) (____/(____)(____)\_)__)

An update client for LastSeen.

Version: %s
Build: %s
`
	// USAGE is the list of valid args available
	USAGE = `
valid arguments:
    -config    - setup the client for use. Running this will re-run the entire login process and overwrite any previous
                config.
    -run       - run the client once. This will check for an existing config file and prompt for one until it exists. 
                 Ctrl+C will get you out.
    -daemon    - once you're happy with the config, use this to launch a daemon that you don't have to worry about.
                 Not a horrible idea to use it in a startup script.
    -debug     - turn on debug mode, requires
    -test <file> - set test file to use; automatically turns on debug    
`
)

type testInfo struct {
	Login loginReq `json:"login"`
	URL   string   `json:"url"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Error       string `json:"error"`
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type pingReq struct {
	Token string `json:"token"`
}

var log = logrus.New()
var debug = false
var testData *testInfo

func main() {

	debug = false

	var config bool
	var run bool
	var signal string
	var testfile string

	flag.BoolVar(&config, "config", false, "setup the client for use")
	flag.BoolVar(&run, "run", false, "run the client once")
	flag.BoolVar(&debug, "debug", false, "turn on debugging")
	flag.StringVar(&testfile, "testfile", "", "file name for test data to use, automatically turns on debug")
	flag.StringVar(&signal, "daemon", "", `send signal to the daemon to:
		start — run/watch in background (runs update on startup)		
		quit — graceful shutdown
		stop — fast shutdown
		reload — reloading the configuration file`)
	flag.Parse()

	/** setup daemon setup things **/
	var daemonize = false
	if signal != "" {
		switch signal {
		case
			"start", "stop", "quit", "reload", "status":
			daemonize = true
			daemon.AddCommand(daemon.StringFlag(&signal, "quit"), syscall.SIGQUIT, termHandler)
			daemon.AddCommand(daemon.StringFlag(&signal, "stop"), syscall.SIGTERM, termHandler)
			daemon.AddCommand(daemon.StringFlag(&signal, "reload"), syscall.SIGHUP, reloadHandler)
			daemon.AddCommand(daemon.StringFlag(&signal, "status"), syscall.Signal(0), statusHandler)
		default:
			log.Errorf("'%s' is not a valid option for -daemon", signal)
			flag.Usage()
			os.Exit(0)
		}

	}
	if daemon.WasReborn() {
		daemonize = true
	}

	/** test file checks **/
	if testfile != "" {
		data, err := ioutil.ReadFile(testfile)
		checkErr(err)
		err = json.Unmarshal(data, &testData)
		checkErr(err)
		debug = true
	}

	/** setup logging **/
	log.Out = os.Stdout
	if debug {
		log.SetLevel(logrus.DebugLevel)
	}

	if (!config && !run && !daemonize) ||
		(config && run) || (run && daemonize) || (config && daemonize) {
		//printUsage("Exactly 1 argument should be passed.")
		if len(os.Args) < 2 {
			log.Errorln("At least 1 argument should be passed")
		} else {
			log.Errorln("One of -config, -run, or -daemon must be provided")
		}
		flag.Usage()
		os.Exit(0)
	}

	if config {
		checkConfig(true)
	} else if run {
		runUpdate()
	} else if daemonize {
		// this is necessary b/c flags are not passed when "reborn"
		// suppose I could get around it with env/config files
		runDaemon()
	} else {
		log.Infoln("whoops, not running anything")
	}

}

func printUsage(err string) {
	fmt.Printf(BANNER, version.VERSION, version.GITCOMMIT)
	fmt.Print(SEP)
	log.Error(err)
	fmt.Print(USAGE)
	os.Exit(0)
}

func writeConfig(resp *http.Response) {
	dataraw, err := ioutil.ReadAll(resp.Body)
	log.Debug("writeConfig:" + string(dataraw))
	checkErr(err)
	var data loginResponse
	err = json.Unmarshal(dataraw, &data)
	checkErr(err)
	if resp.StatusCode == 200 {
		cfgfile, err := homedir.Expand("~/.lastseen/config")
		checkErr(err)
		f, err := os.Create(cfgfile)
		checkErr(err)
		defer f.Close()
		_, err = f.Write(dataraw)
		checkErr(err)
	} else {
		var msg = fmt.Sprintf("Writing config failed: [%v] %v %v", resp.StatusCode, data.Error, data.Message)
		log.Errorln(msg)
		log.Errorln("raw data: " + string(dataraw[:]))
		createConfig()
	}
}

func createConfig() {
	log.Infoln("No config found! Let's create one...")
	prompt := promptui.Prompt{
		Label: "Email",
	}

	if testData != nil {
		prompt.Default = testData.Login.Email
	}

	email, err := prompt.Run()
	checkErr(err)

	log.Infoln("We will NOT save your password")

	prompt = promptui.Prompt{
		Label: "Password",
		Mask:  '*',
	}

	if testData != nil {
		prompt.Default = testData.Login.Password
	}

	pass, err := prompt.Run()
	checkErr(err)

	client := &http.Client{}
	postStruct := loginReq{email, pass}
	postData, err := json.Marshal(postStruct)
	checkErr(err)
	log.Debug("loginReq: " + string(postData))
	prefix := "https://lastseen.me"
	if testData != nil {
		prefix = testData.URL
	}
	req, err := http.NewRequest("POST", prefix+"/api/auth/login", bytes.NewBuffer(postData))
	checkErr(err)
	req.Header.Add("content-type", `application/json"`)
	req.Header.Add("Accept", `application/json"`)
	defer req.Body.Close()

	resp, err := client.Do(req)
	checkErr(err)
	defer resp.Body.Close()

	checkErr(err)
	writeConfig(resp)
}

func checkConfig(create bool) (loginResponse, error) {
	var cfg loginResponse
	cfgfile, err := homedir.Expand("~/.lastseen/config")
	checkErr(err)

	data, err := os.Open(cfgfile)
	if err != nil {

		if create {
			createConfig()
		} else {
			log.Fatal("No config file, please create one first.")
		}
	}

	err = json.NewDecoder(data).Decode(&cfg)

	if err != nil && create {
		createConfig()
	} else {
		checkErr(err)
		if create {
			log.Info("Config appears valid! Try using 'run' to make sure it works")

			prompt := promptui.Prompt{
				Label:     "Create a new config now?",
				IsConfirm: true,
			}

			var input = ""
			for ok := true; ok; ok = (input != "y") {

				fmt.Println("")
				opt, _ := prompt.Run()
				input = opt
				if input == "y" || input == "N" {
					if input == "y" {
						createConfig()
					} else {
						log.Info("Bye!")
					}
					break
				} else {
					log.Error("\nInvalid option - must be 'y' or 'N'")
					continue
				}
			}
		}
	}
	return cfg, nil
}

func runUpdate() {
	cfg, err := checkConfig(false)
	checkErr(err)

	client := &http.Client{}

	postStruct := pingReq{cfg.AccessToken}
	postData, err := json.Marshal(postStruct)
	checkErr(err)
	prefix := "https://lastseen.me"
	if testData != nil {
		prefix = testData.URL
	}

	req, err := http.NewRequest("POST", prefix+"/api/ping", bytes.NewBuffer(postData))
	checkErr(err)
	req.Header.Add("content-type", `application/json"`)
	req.Header.Add("Accept", `application/json"`)

	defer req.Body.Close()

	msgBody := "updated LastSeen!"
	fail := false
	resp, err := client.Do(req)

	if err != nil {
		fail = true
		log.Errorf("ERROR: %s", err)
		msgBody = fmt.Sprintf("Not updated, check the logs (%s)", err)
	} else if resp.StatusCode != 200 {
		fail = true
		body, _ := ioutil.ReadAll(resp.Body)
		log.Errorf("HTTP Status %d : %s", resp.StatusCode, body)
		msgBody = fmt.Sprintf("Not updated, check the logs (%s)", err)
	}

	if fail {
		notify(fmt.Sprintf("Not updated, check the logs (%s)", err), "error")
	} else {
		writeConfig(resp)
		notify(msgBody, "")
		log.Info("updated lastseen")
	}
}

//let's try to send a notification
func notify(body string, icon string) {
	conn, err := dbus.SessionBus()
	checkErr(err)
	if err == nil {
		//See:
		//  https://developer.gnome.org/notification-spec/#command-notify

		appName := ""
		replacesID := uint32(0)
		// gnome icon file names - /usr/share/icons/gnome/24x24/actions/
		appIcon := icon
		summary := "LastSeen"
		body := body
		actions := []string{}
		hints := map[string]dbus.Variant{}
		timeout := int32(5000)

		obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
		call := obj.Call("org.freedesktop.Notifications.Notify", 0, appName, replacesID,
			appIcon, summary, body, actions, hints, timeout)
		if call.Err != nil {
			panic(call.Err)
		}
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func runDaemon() {

	_, err := checkConfig(false)
	checkErr(err)
	/** original log config if I can get it into go-daemon
	  file := lumberjack.Logger{
	      Filename:   logfile,
	      MaxSize:    1, // megabytes
	      MaxBackups: 3,
	      MaxAge:     28,    //days
	      Compress:   false, // disabled by default
	  }
	  checkErr(err)
	  defer func() {
	      err = file.Close()
	      checkErr(err)
	  }()

	  if err == nil {
	      log.Out = &file
	  } else {
	      log.Info("Failed to log to file, using default stderr")
	  }
	*/

	logdir, err := homedir.Expand("~/.lastseen/")
	checkErr(err)
	cntxt := &daemon.Context{
		PidFileName: logdir + "/lastseen.pid",
		PidFilePerm: 0644,
		LogFileName: logdir + "/lastseen_go.log",
		LogFilePerm: 0640,
		WorkDir:     logdir + "/",
		Umask:       027,
		Args:        []string{"lastseen-go -daemon"},
	}

	if len(daemon.ActiveFlags()) > 0 {
		d, err := cntxt.Search()
		if err != nil {
			log.Errorln("Daemon is not running, please start it first.")
			log.Fatalln(err)
		}
		daemon.SendCommands(d)
		return
	}
	/** do this if we're trying to start the daemon **/
	runUpdate()

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatalln(err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()

	log.Info("- - - - - - - - - - - - - - -")
	log.Info("daemon started")

	go worker()

	err = daemon.ServeSignals()
	if err != nil {
		log.Println("Error:", err)
	}
	log.Println("daemon terminated")
}

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func worker() {
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Errorf("Failed to connect to session bus: %s", err)
		log.Errorln("DBus Session is likely not supported without a GUI")
		os.Exit(1)
	}
	runUpdate()
	call := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',interface='org.gnome.ScreenSaver'")
	if call.Err != nil {
		log.Errorf("Failed to add match: %s", call.Err)
		os.Exit(1)
	}
	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	log.Info("Listening to Dbus")

LOOP:
	for {
		time.Sleep(time.Second) // this is work to be done by worker.
		select {
		case <-c:
			for v := range c {
				//&{:1.23 /org/gnome/ScreenSaver org.gnome.ScreenSaver.ActiveChanged [true]}
				//fmt.Println(v)
				if v.Body[0] == false {
					log.Info("screen unlocked, updating in 10 sec")
					time.Sleep(time.Second * 10)
					runUpdate()
				}
				break
			}
		case <-stop:
			log.Info("Disconnecting from Dbus")
			conn.Close()
			conn = nil
			break LOOP
		default:
		}

	}
	done <- struct{}{}
}

func termHandler(sig os.Signal) error {
	log.Info("terminating...")
	stop <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-done
	}
	return daemon.ErrStop
}

func reloadHandler(sig os.Signal) error {
	log.Info("configuration reloaded")
	return nil
}

func statusHandler(sig os.Signal) error {
	//	log.Info(fmt.Printf("signal: %v", sig))
	log.Info("process is running")
	//return daemon.ErrStop
	return nil
}
