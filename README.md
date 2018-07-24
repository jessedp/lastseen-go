## About lastseen-go
[![Build Status](https://api.travis-ci.org/jessedp/lastseen-go.svg?branch=master)](https://travis-ci.org/jessedp/lastseen-go)

A client written in go for [lastseen.me](https://lastseen.me). 
Currently it only runs on the architectures below

You'll need an account to run this, so [register](https://lastseen.me/register) - be sure to note the email/password used

## Installation

#### Binaries
**linux** [386](https://github.com/jessedp/lastseen-go/releases/latest/lastseen-cli-linux-386) / 
[amd64](https://github.com/jessedp/lastseen-go/releases/download/latest/lastseen-cli-linux-amd64) / 
[arm](https://github.com/jessedp/lastseen-go/releases/download/latest/lastseen-cli-linux-arm) / 
[arm64](https://github.com/jessedp/lastseen-go/releases/download/latest/lastseen-cli-linux-arm64)

Download the proper one for your system and put it somewhere in your PATH

#### Via Go

```bash
$ go get github.com/jessedp/lastseen-go
$ cd to source
$ make install
```
## Usage

```conosle
Usage of ./lastseen-go:
  -config
    	setup the client for use
  -daemon string
    	send signal to the daemon to:
    			start — run/watch in background (runs update on startup)		
    			quit — graceful shutdown
    			stop — fast shutdown
    			reload — reloading the configuration file
  -debug
    	turn on debugging
  -run
    	run the client once
```
Make it easy on your self and place the binary in your PATH, otherwise
be sure to use the full path to it.
##### Configuring
1. run `lastseen-go -config`
2. run `lastseen-go -run` to make sure it works

##### Set it and leave it
Add `<path_to_bin>/lastseen-go -daemon start` to something that runs 
at start-up (e.g. your `.bashrc`)

- each time it runs it immediately updates the site
- if you have DBus, every time the screen is unlocked, it will
    update the site

__needless to this to say, running this via a cron job defeats the purpose__

### Todos

- get it to compile without dbus
- see about consistent logging format in the daemon
- swap out `flag` package for something else (docopt, go-flags?)  


### Thanks to...
 - a plethora of scripts from https://github.com/jessfraz
