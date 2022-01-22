/**
 * Toy chat server implementation
 * Needs support for parsing user input and sending/listing messages as such
 * 	@user 	- send to user
 * 	@all	- send to all
 * 	list	- list all online users
 * 	logout	- logout of the chat system
 *
 * Uses RPC
 */
package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"net"
	"net/rpc"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Nothing bool

type Message struct {
	User   string
	Target string
	Msg    string
}

type ChatClient struct {
	Username string
	Address  string
	Client   *rpc.Client
}

func (c *ChatClient) getClientConnection() *rpc.Client {
	var err error

	if c.Client == nil {
		c.Client, err = rpc.DialHTTP("tcp", c.Address)
		if err != nil {
			log.Panicf("Error establishing connection with host: %q", err)
		}
	}

	return c.Client
}

// Register takes a username and registers it with the server
func (c *ChatClient) Register() {
	var reply string
	c.Client = c.getClientConnection()

	err := c.Client.Call("ChatServer.Register", c.Username, &reply)
	if err != nil {
		log.Printf("Error registering user: %q", err)
	} else {
		log.Printf("Reply: %s", reply)
	}
}

// CheckMessages does a check every second for new messages for the user
func (c *ChatClient) CheckMessages() {
	var reply []string
	c.Client = c.getClientConnection()

	for {
		err := c.Client.Call("ChatServer.CheckMessages", c.Username, &reply)
		if err != nil {
			log.Fatalln("Chat has been shutdown. Goodbye.")
		}

		for i := range reply {
			log.Println(reply[i])
		}

		time.Sleep(time.Second)
	}
}

// List lists all the users in the chat currently
func (c *ChatClient) List() {
	var reply []string
	var none Nothing
	c.Client = c.getClientConnection()

	err := c.Client.Call("ChatServer.List", none, &reply)
	if err != nil {
		log.Printf("Error listing users: %q\n", err)
	}

	for i := range reply {
		log.Println(reply[i])
	}
}

// Tell sends a message to a specific user
func (c *ChatClient) Tell(params []string) {
	var reply Nothing
	c.Client = c.getClientConnection()

	if len(params) > 2 {
		msg := strings.Join(params[2:], " ")
		message := Message{
			User:   c.Username,
			Target: params[1],
			Msg:    msg,
		}

		err := c.Client.Call("ChatServer.Tell", message, &reply)
		if err != nil {
			log.Printf("Error telling users something: %q", err)
		}
	} else {
		log.Println("Usage of tell: tell <user> <msg>")
	}
}

// Say sends a message to all users
func (c *ChatClient) Say(params []string) {
	var reply Nothing
	c.Client = c.getClientConnection()

	if len(params) > 1 {
		msg := strings.Join(params[1:], " ")
		message := Message{
			User:   c.Username,
			Target: params[1],
			Msg:    msg,
		}

		err := c.Client.Call("ChatServer.Say", message, &reply)
		if err != nil {
			log.Printf("Error saying something: %q", err)
		}
	} else {
		log.Println("Usage of say: say <msg>")
	}
}

// Logout logs out the current user and shuts down the client
func (c *ChatClient) Logout() {
	var reply Nothing
	c.Client = c.getClientConnection()

	err := c.Client.Call("ChatServer.Logout", c.Username, &reply)
	if err != nil {
		log.Printf("Error logging out: %q", err)
	}
}

// Shutdown stops the server and logs out all users
func (c *ChatClient) Shutdown() {
	var request Nothing = false
	var reply Nothing
	c.Client = c.getClientConnection()

	err := c.Client.Call("ChatServer.Shutdown", request, &reply)
	if err != nil {
		log.Printf("Error shutting down server: %q", err)
	}
}

// Globals/Constants
var (
	DEFAULT_PORT = 3410
	DEFAULT_HOST = "localhost"
)

func createClientFromFlags() (*ChatClient, error) {
	var c *ChatClient = &ChatClient{}
	var host string

	flag.StringVar(&c.Username, "user", "fred", "Your username")
	flag.StringVar(&host, "host", "localhost", "The host you want to connect to")

	flag.Parse()

	if !flag.Parsed() {
		return c, errors.New("Unable to create user from commandline flags. Please try again")
	}

	// Check for the structure of the flag to see if we can make any educated guesses for them
	if len(host) != 0 {

		if strings.HasPrefix(host, ":") { // Begins with a colon means :3410 (just port)
			c.Address = DEFAULT_HOST + host
		} else if strings.Contains(host, ":") { // Contains a colon means host:port
			c.Address = host
		} else { // Otherwise, it's just a host
			c.Address = net.JoinHostPort(host, strconv.Itoa(DEFAULT_PORT))
		}

	} else {
		c.Address = net.JoinHostPort(DEFAULT_HOST, strconv.Itoa(DEFAULT_PORT)) // Default to our default port and host
	}

	return c, nil
}

func mainLoop(c *ChatClient) {
	for {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error: %q\n", err)
		}

		line = strings.TrimSpace(line)
		params := strings.Fields(line)

		if strings.HasPrefix(line, "list") {
			c.List()
		} else if strings.HasPrefix(line, "tell") {
			c.Tell(params)
		} else if strings.HasPrefix(line, "say") {
			c.Say(params)
		} else if strings.HasPrefix(line, "logout") {
			c.Logout()
			break
		} else {
			c.Shutdown()
			break
		}
	}
}

func main() {
	// Set MAX PROCS
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Start by parsing any flags given to the program
	client, err := createClientFromFlags()
	if err != nil {
		log.Panicf("Error creating client from flags: %q", err)
	}

	client.Register()

	// Listen for messages
	go client.CheckMessages()

	mainLoop(client)
}
