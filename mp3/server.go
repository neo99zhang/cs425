package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ARG_NUM_CLIENT int = 2

var wg sync.WaitGroup
var conn net.Conn
var err error

func FindMax(array []int) int {
	var max int = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
	}
	return max
}

type Server struct {
	me        string // self name , e.g. A
	mu_RTS    *sync.Mutex
	mu_TW     *sync.Mutex
	address   map[string](string)
	port      map[string](string)
	accounts  map[string](Account)
	send_conn map[string](net.Conn) // [client's name](connection)
	read_conn map[string](net.Conn) // [client's name](connection)
	TW        map[string][]write_vale
	RTS       map[string][]int
	ln        net.Listener
}

type Operation struct {
	method    string
	branch    string
	account   string
	amount    int
	timestamp int
}

func (op *Operation) Init(operation string) {
	words := strings.Fields(operation)
	op.method = words[0]
	switch op.method {
	case "DEPOSITE":
		if len(words) != 4 {
			panic("DEPOSITE missing info")
		}
		op.branch = words[1]
		amount, err := strconv.Atoi(words[2])
		if err != nil {
			panic(err)
		}
		op.amount = amount
		timestamp, err := strconv.Atoi(words[3])
		if err != nil {
			panic(err)
		}
		op.timestamp = timestamp

	case "WIDTHDRAW":
		if len(words) != 4 {
			panic("WIDTHDRAW missing info")
		}
		op.branch = words[1]
		amount, err := strconv.Atoi(words[2])
		if err != nil {
			panic(err)
		}
		op.amount = -amount
		timestamp, err := strconv.Atoi(words[3])
		if err != nil {
			panic(err)
		}
		op.timestamp = timestamp

	case "BALANCE":
		if len(words) != 3 {
			panic("BALANCE missing info")
		}
		op.branch = words[1]
		if err != nil {
			panic(err)
		}
		timestamp, err := strconv.Atoi(words[2])
		if err != nil {
			panic(err)
		}
		op.timestamp = timestamp

	case "COMMIT":
		timestamp, err := strconv.Atoi(words[1])
		if err != nil {
			panic(err)
		}
		op.timestamp = timestamp

	default:
		panic("The operation is not found.")
	}

}

type write_vale struct {
	timestamp int
	value     int
}

type Account struct {
	mu *sync.RWMutex
	// mu_RTS             *sync.Mutex
	name               string
	committedValue     int
	committedTimestamp int
	// TW                 []write_vale
	// RTS                []int
	amount int
	// temAmount int
}

//check if ac exists in server
func check_inList(ac string, m map[string](Account)) bool {
	for a, _ := range m {
		if strings.Compare(ac, a) == 0 {
			return true
		}
	}
	return false
}

func (sv *Server) read(op Operation) (int, bool) {
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
	abort := false
	if !check_inList(op.account, sv.accounts) {
		abort = true
		return 0, abort
	}

	sv.accounts[op.account].mu.RLock()

	// if (Tc > write timestamp on committed version of D)
	if op.timestamp > sv.accounts[op.account].committedTimestamp {
		// Ds = version of D with the maximum write timestamp that is ≤ Tc
		Ds := sv.accounts[op.account].committedValue
		TS := sv.accounts[op.account].committedTimestamp
		committed := true
		sv.mu_TW.Lock()

		for i := range sv.TW[op.account] {
			if sv.TW[op.account][i].timestamp <= op.timestamp {
				Ds += sv.TW[op.account][i].value
				if sv.TW[op.account][i].timestamp > TS {
					TS = sv.TW[op.account][i].timestamp
				}
				committed = false
			}
		}
		sv.mu_TW.Unlock()

		// if (Ds is committed)
		// read Ds and add Tc to RTS list (if not already added)
		if committed {
			sv.mu_RTS.Lock()
			sv.RTS[op.account] = append(sv.RTS[op.account], op.timestamp)
			sv.mu_RTS.Unlock()
			sv.accounts[op.account].mu.RUnlock()
			return Ds, abort
		} else {
			sv.accounts[op.account].mu.RUnlock()
			// if Ds was written by Tc, simply read Ds
			if op.timestamp == TS {
				return Ds, abort
			} else {
				// wait until the transaction that wrote Ds is committed or aborted, and
				// reapply the read rule.
				flag := true
				for {
					sv.mu_TW.Lock()
					for i := range sv.TW[op.account] {
						if sv.TW[op.account][i].timestamp == TS {
							flag = false
							break
						}
					}
					if flag {
						sv.mu_TW.Unlock()
						return sv.read(op)
					}
					sv.mu_TW.Unlock()
					time.Sleep(20 * time.Millisecond)
				}

			}
		}

		// abort transaction Tc
		// too late; a transaction with later timestamp has already written the object.
	} else {
		abort = true
		sv.accounts[op.account].mu.RUnlock()
		return 0, abort
	}

}

func (sv *Server) write(op Operation) bool {
	abort := false
	// Write Rule
	// Transaction Tc requests a write operation on object D
	sv.accounts[op.account].mu.Lock()
	defer sv.accounts[op.account].mu.Unlock()
	if check_inList(op.account, sv.accounts) == false && op.method == "WITHDRAW" {
		abort = true
		return abort
	}
	// if (Tc ≥ max. read timestamp on D
	// && Tc > write timestamp on committed version of D)
	sv.mu_RTS.Lock()
	defer sv.mu_RTS.Unlock()
	sv.mu_TW.Lock()
	defer sv.mu_TW.Unlock()
	if (len(sv.RTS[op.account]) == 0 || op.timestamp >= FindMax(sv.RTS[op.account])) && (len(sv.TW[op.account]) == 0 || op.timestamp > sv.accounts[op.account].committedTimestamp) {
		// Perform a tentative write on D:
		// If Tc already has an entry in the TW list for D, update it.
		foundFlag := false
		for i := range sv.TW[op.account] {
			if sv.TW[op.account][i].timestamp == op.timestamp {
				foundFlag = true
				// newAccount := Account{
				// 	name:               sv.accounts[op.account].name,
				// 	committedValue:     sv.accounts[op.account].committedValue,
				// 	committedTimestamp: sv.accounts[op.account].committedTimestamp,
				// 	amount:             sv.accounts[op.account].amount + op.amount,
				// }
				// sv.accounts[op.account] = newAccount
				am := 0
				if op.method == "DEPOSIT" {
					am = op.amount
				} else if op.method == "WITHDRAW" {
					am = -1 * op.amount
				}

				// newWriteValue := write_vale{
				// 	timestamp: op.timestamp,
				// 	value:     am,
				// }
				sv.TW[op.account][i].value += am
				break
			}
		}
		if foundFlag == false {
			// Else, add Tc and its write value to the TW list.
			tuple := write_vale{
				timestamp: op.timestamp,
				value:     op.amount,
			}
			if op.method == "WITHDRAW" {
				tuple.value *= -1
			}
			sv.TW[op.account] = append(sv.TW[op.account], tuple)
		}

	} else {
		// else
		// abort transaction Tc
		abort = true
		// //too late; a transaction with later timestamp has already read or
		// written the object.
	}

	return abort

}

//Tell if the conn is from a Client or a server
//if a server return false, its name(by IP)
//if a client return true, empty string
func (sv *Server) judgeClient(conn net.Conn) (bool, string) {
	// reader := bufio.NewReader(conn)
	// // incoming, error1 := reader.ReadBytes('\n')
	// if error1 != nil {
	// 	fmt.Println("Connection error")
	// 	conn.Close()
	// }

	addr := conn.RemoteAddr().String()
	straddr := strings.Split(addr, ":")[0]
	fmt.Println("Connected to addr ", straddr)
	for i := range sv.address {
		if sv.address[i] == straddr {
			fmt.Println("Connected to a server with IP", straddr)
			return false, straddr
		}
	}
	rtstr := ""
	return true, rtstr

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

// Send a message to client
// Notice: Need to set connection before
func (sv *Server) sendtoClient(text string, send_conn net.Conn) {
	fmt.Fprintf(send_conn, "%s\n", text)
}

// Continuously read message from the client
// Notice: Need to set connection before
func (sv *Server) readClient(read_conn net.Conn) string {
	var buf [512]byte
	result := bytes.NewBuffer(nil)
	n, err := read_conn.Read(buf[0:])
	result.Write(buf[0:n])
	if err != nil {
		if err == io.EOF {
			return result.String()
		}
		panic(err)
	}
	return result.String()
}

func (sv *Server) start_listen() {
	ln, err := net.Listen("tcp", strings.Join([]string{sv.address[sv.me], sv.port[sv.me]}, ":"))
	if err != nil {
		panic(err)
	}
	sv.ln = ln
}

// Tryting to connect to the client, once connect, it would return
// the read channel and send channel
func (sv *Server) connect_client() (net.Conn, net.Conn) {
	read_conn, err := sv.ln.Accept()
	if err != nil {
		panic(err)
	}

	addr := conn.RemoteAddr().String()
	clientAddr := strings.Split(addr, ":")[0]
	fmt.Println(clientAddr)
	send_conn, err := net.Dial("tcp", strings.Join([]string{clientAddr, "1023"}, ":"))
	if err != nil {
		panic(err)
	}
	return read_conn, send_conn
}

func (sv *Server) abort(timestamp string) {

	for acc:= range sv.TW {
		for i := 0; i < len(sv.TW[acc]);{
			if timestamp == sv.TW[acc][i].timestamp  {
					sv.TW[acc] = append(sv.TW[acc][:i], sv.TW[acc][i+1:]...)
					// sv.TW[acc][i] = sv.TW[acc][len(sv.TW[acc])-1] // Copy last element to index i.
					// sv.TW[acc][len(sv.TW[acc])-1] = ""   // Erase last element (write zero value).
					// sv.TW[acc] = sv.TW[acc][:len(sv.TW[acc])-1]
			}else {
				i++
			}
		}

	}
	for acc:= range sv.RTS {
		for i := 0; i < len(sv.RTS);{
			if timestamp == sv.RTS[i] {
				sv.RTS[acc] = append(sv.RTS[acc][:i], sv.RTS[acc][i+1:]...)
			}else {
				i++
			}
		}
	}
	//TODO also notify other server to do abort
	//TODO send abort message back
}

func (sv *Server) commit(timestamp int) bool {
	abort := false
	// iterate every account, check whether the final balance is negative
	for name, TW := range sv.TW {
		Ds := sv.accounts[name].committedValue
		for i := range TW {
			if TW[i].timestamp <= timestamp {
				Ds += TW[i].value
			}
		}
		if Ds < 0 {
			// abort the transaction
			sv.abort(timestamp)
			abort = true
		}
	}
	// commit the transaction
	for name, TW := range sv.TW {
		Ds := sv.accounts[name].committedValue
		TS := sv.accounts[name].committedTimestamp
		flag := false
		for i := 0; i <  len(TW);{
			if TW[i].timestamp <= timestamp {
				Ds += TW[i].value
				if TW[i].timestamp == timestamp{
					flag = true
					sv.TW[acc] = append(sv.TW[acc][:i], sv.TW[acc][i+1:]...)
				}
			}else{
				i++
			}
		}
		if flag{
			sv.accounts[name].committedValue = Ds
			sv.accounts[name].committedTimestamp = timestamp 
		}
	}
	sv.

	return abort
}
func (sv *Server) handleOperation(op Operation, send_conn net.Conn) {
	switch op.method {
	case "DEPOSITE":
		abort := sv.write(op)
		if abort {
			sv.sendtoClient("ABORTED", send_conn)
			sv.abort(op.timestamp)
		} else {
			sv.sendtoClient("OK", send_conn)
		}

	case "WIDTHDRAW":
		abort := sv.write(op)
		if abort {
			sv.sendtoClient("ABORTED", send_conn)
			sv.abort(op.timestamp)
		} else {
			sv.sendtoClient("OK", send_conn)
		}

	case "BALANCE":
		value, abort := sv.read(op)
		if abort {
			sv.sendtoClient("ABORTED", send_conn)
			sv.abort(op.timestamp)
		} else {
			sv.sendtoClient(strconv.Itoa(value), send_conn)
		}

	case "COMMIT":
		abort := sv.commit(op.timestamp)
		if abort {
			sv.sendtoClient("ABORTED", send_conn)
			sv.abort(op.timestamp)
		} else {
			sv.sendtoClient("OK", send_conn)
		}
	}
}

func (sv *Server) handleConnection(read_conn net.Conn, send_conn net.Conn) {
	//handle all from one connection(IP)
	//TODO judge branch or client
	operations := sv.readClient(read_conn)
	operations_list := strings.Split(operations, "\n")
	for _, operation := range operations_list {
		op := new(Operation)
		op.Init(operation)
		// TODO handle operation based on the operation types
		sv.handleOperation(op, send_conn)
	}

	defer conn.Close()

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
	// sv.build_branches()	// connect to all other branches
	sv.start_listen()
	for {
		read_conn, send_conn := sv.connect_client() // connect once, always listen self' port
		go handleConnection(read_conn, send_conn)
	}
	// wg.Add(1)
	// go sv.handleClient()
	// sv.sendtoClient("1")
	// wg.Wait()
}
