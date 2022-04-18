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
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

const ARG_NUM_CLIENT int = 2

var wg sync.WaitGroup
var conn net.Conn
var err error

type Server struct {
	me      string // self name , e.g. A
	address map[string](string)
	port    map[string](string)
}

func (sv *Server) readFromConfig(config_file string) {
	// read config line by line
	file, err := os.Open(config_file)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		arr := strings.Split(line, " ")
		fmt.Println(arr[0], arr[1])
		sv.address[arr[0]] = arr[1]
		sv.port[arr[0]] = arr[2]
		fmt.Println(sv.address, sv.port)
	}
}

func handle_transaction() {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		fmt.Fprintf(conn, "%s", text)
	}
}

func main() {
	argv := os.Args[1:]
	if len(argv) != ARG_NUM_CLIENT {
		fmt.Fprintf(os.Stderr, "usage: ./server <Server Name [A,B,C,D,E]> <config.txt>\n")
		os.Exit(1)
	}
	sv := &Server{}
	sv.me = argv[0]
	sv.address = make(map[string]string)
	sv.port = make(map[string]string)
	config_file := argv[1]

	sv.readFromConfig(config_file)

	// file.Close()

	// conn, err = net.Dial("tcp", strings.Join([]string{serv_addr, serv_port}, ":"))

	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "%s\n", err)
	// 	os.Exit(1)
	// }

	// fmt.Fprintf(conn, "%s\n", node_name)

	// wg.Add(1)
	// go handle_transaction()
	// wg.Wait()
}
