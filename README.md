## About lastseen-go
[![Build Status](https://api.travis-ci.org/jessedp/lastseen-go.svg?branch=master)](https://travis-ci.org/jessedp/lastseen-go)

A client written in go for [lastseen.me](https://lastseen.me]

You'll need an account to run this, so [register](https://lastseen.me/register] first (note the email/password used)

## Installation

#### Binaries
- **linux** [386](https://github.com/jessedp/lastseen-go/releases/download/v0.1.0/lastseen-cli-linux-386) / [amd64](https://github.com/jessedp/lastseen-go/releases/download/v0.1.0/lastseen-cli-linux-amd64) / [arm](https://github.com/jessedp/lastseen-go/releases/download/v0.1.0/lastseen-cli-linux-arm) / [arm64](https://github.com/jessedp/lastseen-go/releases/download/v0.1.0/lastseen-cli-linux-arm64)

#### Via Go

```bash
$ go get github.com/jessedp/lastseen-go
```
## Usage

```conosle
 __     __   ____  ____  ____  ____  ____  __ _
(  )   / _\ / ___)(_  _)/ ___)(  __)(  __)(  ( \
/ (_/\/    \\___ \  )(  \___ \ ) _)  ) _) /    /
\____/\_/\_/(____/ (__) (____/(____)(____)\_)__)

An update client for LastSeen.

-------------------------------------------------------------------
Exactly 1 argument should be passed.

valid arguments:
    config    - setup the client for use. Running this will re-run the entire login process and overwrite any
                previous config.
    run       - run the client once. This will check for an existing config file and prompt for one until it
                exists.
                Ctrl+C will get you out.

    service/daemon options:
    daemon    - once you're happy with the config, use this to launch a daemon that you don't have to worry
                about.
                Not a horrible idea to use it in a startup script.
```

### With a GUI (gnome, most window managers, etc.)
1. run `config`
2. run `run` to make sure it works
3. add `lastseen-cli daemon` to something that runs at start up (e.g. your `.bashrc`)

### Withoutt Dbus (you're using a GUI, window manager, etc.)
1. run `config`
2. run `run` to make sure it works
3. add `lastseen-cli run` to something that runs when you create a new shell (e.g. your `.bashrc`)

__so adding this to, say, a cron job defeats the purpose__


# Thanks to...
I re-used a bunch of scripts from @jessfraz

