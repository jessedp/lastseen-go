package main

import (
    "fmt"
    "net/http"
    "os"
    "io/ioutil"
    "bytes"
    "encoding/json"

    "github.com/atrox/homedir"
    "github.com/manifoldco/promptui"
    "github.com/sirupsen/logrus"
    "github.com/godbus/dbus"
    "github.com/jessedp/lastseen-go/version"
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
    config    - setup the client for use. Running this will re-run the entire login process and overwrite any previous
                config.
    run       - run the client once. This will check for an existing config file and prompt for one until it exists. 
                Ctrl+C will get you out.

    service/daemon options:
    daemon    - once you're happy with the config, use this to launch a daemon that you don't have to worry about.
                Not a horrible idea to use it in a startup script.
`
)

type loginReq struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type loginResponse struct {
    Error       string `json:"error"`
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    ExpiresIn   int64  `json:"expires_in"`
}

type pingReq struct {
    Token string `json:"token"`
}

var log = logrus.New()

func main() {
    args := os.Args[1:]
    if len(args) != 1 {
        printUsage("Exactly 1 argument should be passed.")
    }
    validArgs := map[string]bool{
        "config": true,
        "run":    true,
        "daemon": true,
    }
    theArg := args[0]
    if !validArgs[theArg] {
        printUsage(fmt.Sprintf("%s is not a valid parameter.", theArg))
    } else {
        log.Out = os.Stdout
        switch theArg {
        case "config":
            checkConfig(true)
        case "run":
            runUpdate()
        case "daemon":
            runDaemon()
        }
    }

}

func printUsage(err string) {
    fmt.Printf(BANNER, version.VERSION, version.GITCOMMIT)
    fmt.Print(SEP)
    fmt.Println(err)
    fmt.Print(USAGE)
    os.Exit(1)
}

func writeConfig(resp *http.Response) {
    dataraw, err := ioutil.ReadAll(resp.Body)
    var data loginResponse
    err = json.Unmarshal(dataraw, &data)
    checkErr(err)
    if resp.StatusCode == 200 {
        cfgfile, err := homedir.Expand("~/.lastseen/config")
        f, err := os.Create(cfgfile)
        checkErr(err)
        defer f.Close()
        _, err = f.Write(dataraw)
        checkErr(err)
    } else {
        log.Errorln("That didn't work, the server said: " + data.Error)
        createConfig()
    }

}
func createConfig() {
    log.Infoln("No config found! Let's create one...")
    prompt := promptui.Prompt{
        Label: "Email",
    }

    email, err := prompt.Run()
    checkErr(err)

    log.Infoln("We will NOT save your password")
    prompt = promptui.Prompt{
        Label: "Password",
    }

    pass, err := prompt.Run()
    checkErr(err)

    client := &http.Client{}
    postStruct := loginReq{email, pass}
    postData, err := json.Marshal(postStruct)
    checkErr(err)
    //fmt.Println(bytes.NewBuffer(postData))
    req, err := http.NewRequest("POST", "https://lastseen.me/api/auth/login", bytes.NewBuffer(postData))
    req.Header.Add("content-type", `application/json"`)
    req.Header.Add("Accept", `application/json"`)
    defer req.Body.Close()

    resp, err := client.Do(req)
    writeConfig(resp)
}

func checkConfig(create bool) (loginResponse, error) {
    cfgfile, err := homedir.Expand("~/.lastseen/config")
    checkErr(err)

    data, err := os.Open(cfgfile)
    var cfg loginResponse
    err = json.NewDecoder(data).Decode(&cfg)

    if err != nil && create {
        createConfig()
    } else if err != nil {
        checkErr(err)
    } else {
        if create {
            log.Info("Config appears valid! Try using 'run' to make sure it works")
        }
    }
    return cfg, nil
}

func runUpdate() {
    cfg, err := checkConfig(false)
    checkErr(err)

    log.Info("updating lastseen")

    client := &http.Client{}

    postStruct := pingReq{cfg.AccessToken}
    postData, err := json.Marshal(postStruct)
    checkErr(err)
    //    fmt.Println(bytes.NewBuffer(postData))
    req, err := http.NewRequest("POST", "https://lastseen.me/api/ping", bytes.NewBuffer(postData))
    req.Header.Add("content-type", `application/json"`)
    req.Header.Add("Accept", `application/json"`)
    defer req.Body.Close()

    resp, err := client.Do(req)

    writeConfig(resp)
    //let's try to send a notification
    conn, err := dbus.SessionBus()
    if err == nil {
        obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
        call := obj.Call("org.freedesktop.Notifications.Notify", 0, "", uint32(0),
            "", "LastSeen", "updated LastSeen!", []string{},
            map[string]dbus.Variant{}, int32(5000))
        if call.Err != nil {
            panic(call.Err)
        }
    }
    log.Info("updated lastseen")

}

func runDaemon() {

    logfile, err := homedir.Expand("~/.lastseen/lastseen_go.log")
    file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY, 0666)
    defer file.Close()
    if err == nil {
        log.Out = file
    } else {
        log.Info("Failed to log to file, using default stderr")
    }

    log.Info("starting daemon")
    _, err = checkConfig(false)
    checkErr(err)

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
    conn.Signal(c);
    for v := range c {
        //&{:1.23 /org/gnome/ScreenSaver org.gnome.ScreenSaver.ActiveChanged [true]}
        //fmt.Println(v)
        if v.Body[0] == true {
            log.Info("screen unlocked, running")
            runUpdate()
        }
    }

}

func checkErr(err error) {
    if err != nil {
        log.Fatal(err)
    }
}
