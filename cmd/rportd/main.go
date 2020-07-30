package main

import (
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"

	chserver "github.com/cloudradar-monitoring/rport/server"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

var serverHelp = `
  Usage: rportd [options]

  Options:

    --listen, Defines the IP address the HTTP server listens on.
    (defaults to the environment variable RPORT_LISTEN and falls back to 0.0.0.0).

    --port, -p, Defines the HTTP listening port (defaults to the environment
    variable RPORT_PORT and fallsback to port 8080).

    --key, An optional string to seed the generation of a ECDSA public
    and private key pair. All communications will be secured using this
    key pair. Share the subsequent fingerprint with clients to enable detection
    of man-in-the-middle attacks (defaults to the RPORT_KEY environment
    variable, otherwise a new key is generate each run).

    --authfile, An optional path to a users.json file. This file should
    be an object with users defined like:
      {
        "<user:pass>": ["<addr-regex>","<addr-regex>"]
      }
    when <user> connects, their <pass> will be verified and then
    each of the remote addresses will be compared against the list
    of address regular expressions for a match.

    --auth, An optional string representing a single user with full
    access, in the form of <user:pass>. This is equivalent to creating an
    authfile with {"<user:pass>": [""]}.

    --proxy, Specifies another HTTP server to proxy requests to when
    rport receives a normal HTTP request. Useful for hiding rport in
    plain sight.

    --api-addr, Defines the IP address and port the API server listens on.
    e.g. "0.0.0.0:7777". (defaults to the environment variable RPORT_API_ADDR
    and fallsback to empty string: API not available)

    --doc-root, Specifies local directory path. If specified, rportd will serve
    files from this directory on the same API address (--api-addr).

    --api-auth, Defines "user:password"" authentication pair for accessing API
    e.g. "admin:1234". (defaults to the environment variable RPORT_API_AUTH
    and fallsback to empty string: authorization not required).

    --api-jwt-secret, Defines JWT secret used to generate new tokens.
    (defaults to the environment variable RPORT_AUTH_JWT_SECRET and fallsback
    to auto-generated value).

    -v, Enable verbose logging

    --help, This help text

    --version, Print version info and exit

  Signals:
    The rportd process is listening for SIGUSR2 to print process stats

`

func main() {
	listenInterface := flag.String("listen", "", "")
	p := flag.String("p", "", "")
	port := flag.String("port", "", "")
	key := flag.String("key", "", "")
	authfile := flag.String("authfile", "", "")
	auth := flag.String("auth", "", "")
	proxy := flag.String("proxy", "", "")
	apiAddr := flag.String("api-addr", "", "")
	apiAuth := flag.String("api-auth", "", "")
	apiJWTSecret := flag.String("api-jwt-secret", "", "")
	docRoot := flag.String("doc-root", "", "")
	verbose := flag.Bool("v", false, "")
	version := flag.Bool("version", false, "")

	flag.Usage = func() {
		fmt.Print(serverHelp)
		os.Exit(1)
	}
	flag.Parse()

	if *version {
		fmt.Println(chshare.BuildVersion)
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		fmt.Println("Unsupported command or argument.")
		fmt.Print(serverHelp)
		os.Exit(1)
	}

	if *listenInterface == "" {
		*listenInterface = os.Getenv("RPORT_LISTEN")
	}
	if *listenInterface == "" {
		*listenInterface = "0.0.0.0"
	}
	if *port == "" {
		*port = *p
	}
	if *port == "" {
		*port = os.Getenv("RPORT_PORT")
	}
	if *port == "" {
		*port = "8080"
	}
	if *key == "" {
		*key = os.Getenv("RPORT_KEY")
	}
	if *apiAddr == "" {
		*apiAddr = os.Getenv("RPORT_API_ADDR")
	}
	if *apiAuth == "" {
		*apiAuth = os.Getenv("RPORT_API_AUTH")
	}
	if *apiJWTSecret == "" {
		*apiJWTSecret = os.Getenv("RPORT_API_JWT_SECRET")
	}
	if *apiJWTSecret == "" {
		*apiJWTSecret = generateJWTSecret()
	}

	if *docRoot != "" && *apiAddr == "" {
		log.Fatal("To use --doc-root you need to specify API address (see --api-addr)")
	}

	s, err := chserver.NewServer(&chserver.Config{
		KeySeed:      *key,
		AuthFile:     *authfile,
		Auth:         *auth,
		Proxy:        *proxy,
		Verbose:      *verbose,
		APIAuth:      *apiAuth,
		APIJWTSecret: *apiJWTSecret,
		DocRoot:      *docRoot,
	})
	if err != nil {
		log.Fatal(err)
	}

	go chshare.GoStats()

	if err = s.Run(*listenInterface+":"+*port, *apiAddr); err != nil {
		log.Fatal(err)
	}
}

func generateJWTSecret() string {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		log.Fatalf("can't generate API JWT secret: %s", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}