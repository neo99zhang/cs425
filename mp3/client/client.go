// Write Rule
// Transaction Tc requests a write operation on object D
// if (Tc ≥ max. read timestamp on D
// && Tc > write timestamp on committed version of D)
// Perform a tentative write on D:
// If Tc already has an entry in the TW list for D, update it.
// Else, add Tc and its write value to the TW list.
// else
// abort transaction Tc
// //too late; a transaction with later timestamp has already read or
// written the object.

// //Read Rule
// Transaction Tc requests a read operation on object D
// if (Tc > write timestamp on committed version of D) {
// Ds = version of D with the maximum write timestamp that is ≤ Tc
// //search across the committed timestamp and the TW list for object D.
// if (Ds is committed)
// read Ds and add Tc to RTS list (if not already added)
// else
// if Ds was written by Tc, simply read Ds
// else
// wait until the transaction that wrote Ds is committed or aborted, and
// reapply the read rule.
// // if the transaction is committed, Tc will read its value after the wait.
// // if the transaction is aborted, Tc will read the value from an older
// transaction.
// } else
// abort transaction Tc
// //too late; a transaction with later timestamp has already written the object.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ARG_NUM_CLIENT int = 2
	// SERVERS = [A B C D E]
)

var wg sync.WaitGroup
var conn net.Conn
var err error

type Client struct {
	me                 string // self name , e.g. A
	address            map[string](string)
	port               map[string](string)
	send_conn          net.Conn
	read_conn          net.Conn
	currentTransaction bool
	stmsp              string
}

func (cl *Client) readFromConfig(config_file string) {
	// read config line by line
	file, err := os.Open(config_file)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		arr := strings.Split(line, " ")
		cl.address[arr[0]] = arr[1]
		cl.port[arr[0]] = arr[2]
		// fmt.Println(cl.address, cl.port)
	}
}

func random() string {

	var list = []string{"A", "B", "C", "D", "E"}
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(100)
	selected := list[num%5]
	//TODO for test return A
	return selected
	// return "A"
}

func (cl *Client) connect_server() {
	coordinator := random()
	// fmt.Println("choose the coordinator: ", coordinator)
	for {
		conn, err := net.Dial("tcp", strings.Join([]string{cl.address[coordinator], cl.port[coordinator]}, ":"))

		if err == nil {
			// fmt.Println("Connected to coordinator")
			cl.send_conn = conn
			break
		}
		// fmt.Println("Conn closed by server and try again")
		time.Sleep(20 * time.Millisecond)
	}
	
	
	ln, err := net.Listen("tcp", strings.Join([]string{"127.0.0.1", "10050"}, ":"))
	
	if err != nil {
		panic(err)
	}
	// fmt.Println("before listen")
	read_conn, err := ln.Accept()
	// fmt.Println("after listen")
	if err != nil {
		panic(err)
	}
	cl.read_conn = read_conn
}

func (cl *Client) wait_response() {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		fmt.Fprintf(conn, "%s", text)
	}
}

func (cl *Client) handleServer() {
	var buf [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := cl.read_conn.Read(buf[0:])
		result.Write(buf[0:n])
		if err != nil {
			// fmt.Println(err)
			// if err == io.EOF {
			// 	break
			// }
			return
		}
		// fmt.Println(result)
	}

}

func main() {
	argv := os.Args[1:]
	if len(argv) != ARG_NUM_CLIENT {
		fmt.Fprintf(os.Stderr, "usage: ./Client <id> <config.txt>\n")
		os.Exit(1)
	}
	cl := Client{
		me:      argv[0],
		address: make(map[string]string),
		port:    make(map[string]string),
	}

	config_file := argv[1]
	cl.readFromConfig(config_file)
	cl.connect_server()
	// cl.wait_response()
	// go fmt.Fprintf(cl.send_conn, "Msg from client: Hello, my name is %s \n", cl.me)

	//TODO make connections to servers
	// wg.Add(1)
	// fmt.Println("before reader")
	reader := bufio.NewReader(os.Stdin)
	cl.currentTransaction = true
	for {
		// fmt.Println("begin for")
		if cl.currentTransaction == false {
			break
		}
		// fmt.Println("get input from reader")
		input, _ := reader.ReadString('\n')
		if len(input) == 0 {
			// fmt.Println("get zero length input")
			continue
		}
		
		input = strings.TrimSpace(input)
		// fmt.Println(input)
		// fmt.Println("get input.txt")
		if "BEGIN" == input { // trimmed to the last before \n
			// if cl.currentTransaction {
			// 	continue
			// }
			//init timestamp and store it in client struct
			tmsp := time.Now().UnixNano()
			cl.stmsp = strconv.FormatInt(tmsp, 10)
			fmt.Println("OK")
		} else {
			if cl.currentTransaction {
				toServer := input + " " + cl.stmsp
				// fmt.Println(toServer)
				fmt.Fprintf(cl.send_conn, "%s\n", toServer)
				reader := bufio.NewReader(cl.read_conn)
				response, error1 := reader.ReadString('\n')
				if error1 != nil {
					fmt.Println("Have error when listening for ", cl.read_conn)
				} else {
					response = strings.TrimSpace(response)
					fmt.Println(response)
					if response == "NOT FOUND, ABORTED" || response == "ABORTED" || response == "COMMIT OK" {
						cl.currentTransaction = false
						// fmt.Println("Debug Message: Ends at ", response)
					}
				}
			}
		}
	}
	// go cl.handleServer()
	// wg.Wait()
}
