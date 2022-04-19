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
	"io"
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
	me        string // self name , e.g. A
	address   map[string](string)
	port      map[string](string)
	accounts  map[string](Account)
	send_conn net.Conn
	read_conn net.Conn
}

type Transaction struct {
	method  string
	branch  string
	account string
	amount  int
}

type Account struct {
	name               string
	committedValue     int
	committedTimestamp int
	TW                 []int
	RTS                []int
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
		sv.address[arr[0]] = arr[1]
		sv.port[arr[0]] = arr[2]
	}
}

func (sv *Server) handleClient() {
	var buf [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := sv.read_conn.Read(buf[0:])
		result.Write(buf[0:n])
		if err != nil {
			fmt.Println(err)
			if err == io.EOF {
				break
			}
			return
		}
		fmt.Println(result)
	}
	fmt.Println(result.Bytes())
}

func (sv *Server) connect_client() {
	ln, err := net.Listen("tcp", strings.Join([]string{sv.address[sv.me], sv.port[sv.me]}, ":"))
	if err != nil {
		panic(err)
	}
	conn, err := ln.Accept()
	if err != nil {
		panic(err)
	}
	sv.read_conn = conn

	addr := conn.RemoteAddr().String()
	clientAddr := strings.Split(addr, ":")[0]
	fmt.Println(clientAddr)
	send_conn, err := net.Dial("tcp", strings.Join([]string{clientAddr, "1023"}, ":"))
	if err != nil {
		panic(err)
	}
	sv.send_conn = send_conn

}
func main() {
	argv := os.Args[1:]
	if len(argv) != ARG_NUM_CLIENT {
		fmt.Fprintf(os.Stderr, "usage: ./server <Server Name [A,B,C,D,E]> <config.txt>\n")
		os.Exit(1)
	}
	sv := Server{
		me:       argv[0],
		address:  make(map[string]string),
		port:     make(map[string]string),
		accounts: make(map[string]Account),
	}
	config_file := argv[1]

	sv.readFromConfig(config_file)
	sv.connect_client()
	sv.handleClient()
}
