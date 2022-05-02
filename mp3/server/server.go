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

const ARG_NUM_SERVER int = 2

var lookup = map[string]int{"AB": 10012, "AC": 10013, "AD": 10014, "AE": 10015,
	"BA": 10021, "BC": 10023, "BD": 10024, "BE": 10025,
	"CA": 10031, "CB": 10032, "CD": 10034, "CE": 10035,
	"DA": 10041, "DB": 10042, "DC": 10043, "DE": 10045,
	"EA": 10051, "EB": 10052, "EC": 10053, "ED": 10054}

// var wg sync.WaitGroup
// var conn net.Conn
var wg sync.WaitGroup

func FindMax(array []int64) int64 {
	var max int64 = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
	}
	return max
}

type Server struct {
	me          string // self name , e.g. A
	mu_RTS      *sync.Mutex
	mu_TW       *sync.Mutex
	mu_commit   *sync.Mutex
	address     map[string](string)
	port        map[string](string)
	accounts    map[string](Account)
	send_conn   map[string](net.Conn) // [server or client's name](connection)
	read_conn   map[string](net.Conn) // [server or client's name](connection)
	TW          map[string][]write_vale
	RTS         map[string][]int64
	ln          net.Listener
	name        map[IP](string)
	commitCount map[string]int
}

type Operation struct {
	method    string
	branch    string
	account   string
	amount    int
	timestamp int64
}

type write_vale struct {
	timestamp int64
	value     int
}

type Account struct {
	mu *sync.RWMutex
	// mu_RTS             *sync.Mutex
	name               string
	committedValue     int
	committedTimestamp int64
	// TW                 []write_vale
	// RTS                []int
	amount           int
	createdTimestamp int64
	// temAmount int
}

func (op *Operation) Init(operation string) {
	words := strings.Fields(operation)
	op.method = words[0]
	switch op.method {
	case "DEPOSIT":
		if len(words) != 4 {
			panic("DEPOSITE missing info")
		}
		op.branch = strings.Split(words[1], ".")[0]
		op.account = strings.Split(words[1], ".")[1]
		amount, err := strconv.Atoi(words[2])
		if err != nil {
			panic(err)
		}
		op.amount = amount
		timestamp, err := strconv.ParseInt(words[3], 10, 64)
		if err != nil {
			panic(err)
		}
		op.timestamp = timestamp

	case "WITHDRAW":
		if len(words) != 4 {
			panic("WIDTHDRAW missing info")
		}
		op.branch = strings.Split(words[1], ".")[0]
		op.account = strings.Split(words[1], ".")[1]
		amount, err := strconv.Atoi(words[2])
		if err != nil {
			panic(err)
		}
		op.amount = -amount
		timestamp, err := strconv.ParseInt(words[3], 10, 64)
		if err != nil {
			panic(err)
		}
		op.timestamp = timestamp

	case "BALANCE":
		if len(words) != 3 {
			panic("BALANCE missing info")
		}
		op.branch = strings.Split(words[1], ".")[0]
		op.account = strings.Split(words[1], ".")[1]
		timestamp, err := strconv.ParseInt(words[2], 10, 64)
		if err != nil {
			panic(err)
		}
		op.timestamp = timestamp

	case "COMMIT":
		timestamp, err := strconv.ParseInt(words[1], 10, 64)
		if err != nil {
			panic(err)
		}
		op.timestamp = timestamp

	default:
		panic("The operation is not found.")
	}

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

	if check_inList(op.account, sv.accounts) == false {
		if op.method == "WITHDRAW" {
			abort = true
			return abort
		} else if op.method == "DEPOSIT" {
			newAccount := Account{
				mu:                 &(make([]sync.RWMutex, 1)[0]),
				name:               op.account,
				committedValue:     0,
				committedTimestamp: 0,
				amount:             op.amount,
				createdTimestamp:   op.timestamp,
			}
			sv.accounts[op.account] = newAccount
		}

	}
	sv.accounts[op.account].mu.Lock()
	defer sv.accounts[op.account].mu.Unlock()
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
				am := op.amount
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
	// straddr := strings.Split(addr, ":")[0]
	strport := strings.Split(addr, ":")[1]
	intport, err := strconv.Atoi(strport)
	if err != nil {
		panic(err)
	}
	// intport -= PORT_DELTA
	// strport = strconv.Itoa(intport)
	// ip := IP{straddr, strport}
	// fmt.Println("Connected to addr ", ip)

	for names, port := range lookup {
		if port == intport {
			strRes := names[0:1]
			// fmt.Println("Connected to a server with name ", strRes)
			return false, strRes
		}
	}
	rtstr := ""
	return true, rtstr

}

type IP struct {
	X, Y string
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
		sv.name[IP{arr[1], arr[2]}] = arr[0]
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
	ln, _ := net.Listen("tcp", strings.Join([]string{sv.address[sv.me], sv.port[sv.me]}, ":"))

	// fmt.Println("Start listening on port: ", sv.port[sv.me])
	sv.ln = ln
}

// Tryting to connect to the client, once connect, it would return
// the read channel and send channel
func (sv *Server) connect_client() (net.Conn, net.Conn) {
	// fmt.Println("before get client connection")
	read_conn, err := sv.ln.Accept()
	// fmt.Println("after get client connection")
	if err != nil {
		panic(err)
	}

	addr := read_conn.RemoteAddr().String()
	clientAddr := strings.Split(addr, ":")[0]
	// fmt.Println(clientAddr)
	var send_conn net.Conn
	var err1 error
	for {
		send_conn, err = net.Dial("tcp", strings.Join([]string{clientAddr, "10050"}, ":"))
		if err1 == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	// send_conn, err := net.Dial("tcp", strings.Join([]string{clientAddr, "1235"}, ":"))
	// if err != nil {
	// 	panic(err)
	// }
	return read_conn, send_conn
}

func (sv *Server) DoAbort(timestamp int64) {

	for acc := range sv.TW {
		for i := 0; i < len(sv.TW[acc]); {
			if timestamp == sv.TW[acc][i].timestamp {
				sv.TW[acc] = append(sv.TW[acc][:i], sv.TW[acc][i+1:]...)
				// sv.TW[acc][i] = sv.TW[acc][len(sv.TW[acc])-1] // Copy last element to index i.
				// sv.TW[acc][len(sv.TW[acc])-1] = ""   // Erase last element (write zero value).
				// sv.TW[acc] = sv.TW[acc][:len(sv.TW[acc])-1]
			} else {
				i++
			}
		}

	}
	for acc := range sv.RTS {
		for i := 0; i < len(sv.RTS); {
			if timestamp == sv.RTS[acc][i] {
				sv.RTS[acc] = append(sv.RTS[acc][:i], sv.RTS[acc][i+1:]...)
			} else {
				i++
			}
		}
	}
	// delete the accounts that were created in this transaction
	for account, value := range sv.accounts {
		if value.createdTimestamp == timestamp {
			delete(sv.accounts, account)
		}
	}
	//TODO also notify other server to do abort
	//TODO send abort message back
}

func (sv *Server) CheckCommit(timestamp int64) bool {
	CanCommit := true
	TxRelated := false
	// iterate every account, check whether the final balance is negative
	sv.mu_TW.Lock()

	for _, TW := range sv.TW {
		for i := range TW {
			if TW[i].timestamp == timestamp {
				TxRelated = true
				break
			}
		}
	}
	if TxRelated {
		for name, TW := range sv.TW {
			Ds := sv.accounts[name].committedValue
			for i := range TW {
				if TW[i].timestamp <= timestamp {
					Ds += TW[i].value

				}
			}
			if Ds < 0 {
				// abort the transaction
				sv.mu_TW.Unlock()
				sv.DoAbort(timestamp)
				CanCommit = false
				return CanCommit
			}
		}
	}

	sv.mu_TW.Unlock()
	return CanCommit
}

func (sv *Server) DoCommit(timestamp int64) {
	// commit the transaction
	sv.mu_TW.Lock()
	defer sv.mu_TW.Unlock()
	for name := range sv.TW {
		Ds := sv.accounts[name].committedValue
		flag := false
		for i := 0; i < len(sv.TW[name]); {
			if sv.TW[name][i].timestamp <= timestamp {
				Ds += sv.TW[name][i].value
				if sv.TW[name][i].timestamp == timestamp {
					flag = true
					sv.TW[name] = append(sv.TW[name][:i], sv.TW[name][i+1:]...)
				} else {
					i++
				}
			}
		}
		if flag {
			sv.accounts[name].mu.Lock()
			defer sv.accounts[name].mu.Unlock()
			newAccount := Account{
				mu:                 &(make([]sync.RWMutex, 1)[0]),
				name:               sv.accounts[name].name,
				committedValue:     Ds,
				committedTimestamp: timestamp,
				amount:             sv.accounts[name].amount,
				createdTimestamp:   sv.accounts[name].createdTimestamp,
			}
			sv.accounts[name] = newAccount
		}
	}
	account_info := "Accounts Info: "
	for account, value := range sv.accounts {
		if value.committedValue > 0 {
			account_info += account + " " + strconv.Itoa(value.committedValue)
		}
	}
	fmt.Println(account_info)

}
func (sv *Server) handleOperation(op Operation, read_conn net.Conn, send_conn net.Conn) bool {
	abort := false

	switch op.method {
	case "DEPOSIT":
		abort = sv.write(op)
		if abort {
			sv.sendtoClient("ABORTED", send_conn)
			sv.DoAbort(op.timestamp)
		} else {
			sv.sendtoClient("OK", send_conn)
		}

	case "WITHDRAW":
		abort = sv.write(op)
		if abort {
			sv.sendtoClient("NOT FOUND, ABORTED", send_conn)
			sv.DoAbort(op.timestamp)
		} else {
			sv.sendtoClient("OK", send_conn)
		}

	case "BALANCE":
		value, aborted := sv.read(op)
		abort = aborted
		if abort {
			sv.sendtoClient("NOT FOUND, ABORTED", send_conn)
			sv.DoAbort(op.timestamp)
		} else {
			sv.sendtoClient(op.branch+"."+op.account+" = "+strconv.Itoa(value), send_conn)
		}

	case "COMMIT":
		strtmsp := strconv.FormatInt(op.timestamp, 10)
		sv.mu_commit.Lock()
		sv.commitCount[strtmsp] = 0
		sv.mu_commit.Unlock()
		abort = !(sv.CheckCommit(op.timestamp))
		if abort {
			sv.sendtoClient("ABORTED", send_conn)
			msg := strings.Join([]string{"ABORT_Coordinator", strtmsp}, " ")
			for _, name := range sv.name {
				if name != sv.me {
					send_conn := sv.send_conn[name]
					sv.sendtoClient(msg, send_conn)
				}

			}
			sv.DoAbort(op.timestamp)
		} else {
			msg := strings.Join([]string{"COMMIT_PREPARE", strtmsp}, " ")
			for _, name := range sv.name {
				if name != sv.me {
					send_conn := sv.send_conn[name]
					sv.sendtoClient(msg, send_conn)
				}
			}
		}
	}
	return abort
}

func (sv *Server) handleConnection(read_conn net.Conn, send_conn net.Conn) {
	//handle all from one connection(IP)
	//TODO judge branch or client
	for {
		operation := sv.readClient(read_conn)
		operation = strings.TrimSpace(operation)
		op := new(Operation)
		op.Init(operation)
		// Add the client's connection
		sv.read_conn[strconv.FormatInt(op.timestamp, 10)] = read_conn
		sv.send_conn[strconv.FormatInt(op.timestamp, 10)] = send_conn
		// TODO handle operation based on the operation types
		if op.branch == sv.me || op.method == "COMMIT" {
			abort := sv.handleOperation(*op, read_conn, send_conn)
			if abort {
				read_conn.Close()
				send_conn.Close()
				break
			} else if op.method == "COMMIT" {
				break
			}
		} else { //if not self account, send to others
			fmt.Fprintf(sv.send_conn[op.branch], "%s\n", operation)
		}

	}

}

// send connection to all other branches
func (sv *Server) build_branches() {
	wg.Add(4)
	for name := range sv.address {
		go func(name string) {
			if name != sv.me {
				dialer := net.Dialer{
					LocalAddr: &net.TCPAddr{
						IP:   net.ParseIP(sv.address[sv.me]),
						Port: lookup[sv.me+name],
					},
				}
				var send_conn net.Conn
				var err error
				for {
					send_conn, err = dialer.Dial("tcp", strings.Join([]string{sv.address[name], sv.port[name]}, ":"))
					if err == nil {
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
				sv.send_conn[name] = send_conn
				wg.Done()
			}
		}(name)

	}

	count := 0
	for {
		
		// fmt.Println("waiting")
		read_conn, err := sv.ln.Accept()

		if err != nil {
			panic(err)
		}

		is_client, name := sv.judgeClient(read_conn)
		if is_client {
			// fmt.Println("Get client but closed it")
			read_conn.Close()
		} else {
			count += 1
			sv.read_conn[name] = read_conn
			// fmt.Println("Connected to branch: ", name)
		}

		if count == 4 {
			// fmt.Println("Connected to all other branch!")
			break
		}
	}
	wg.Wait()
}

func (sv *Server) handleBranch(name string, read_conn net.Conn, send_conn net.Conn) {
	for {
		abort := false
		operation := sv.readClient(read_conn)
		operation = strings.TrimSpace(operation)
		words := strings.Fields(operation)
		switch words[0] {
		case "DEPOSIT":
			op := new(Operation)
			op.Init(operation)
			abort = sv.write(*op)
			if abort {
				msg := strings.Join([]string{"ABORTED", strconv.FormatInt(op.timestamp, 10)}, " ")
				sv.sendtoClient(msg, send_conn)
				sv.DoAbort(op.timestamp)
			} else {
				msg := strings.Join([]string{"OK", strconv.FormatInt(op.timestamp, 10)}, " ")
				sv.sendtoClient(msg, send_conn)
			}

		case "WITHDRAW":
			op := new(Operation)
			op.Init(operation)
			abort = sv.write(*op)
			if abort {
				msg := strings.Join([]string{"NOT", strconv.FormatInt(op.timestamp, 10)}, " ")
				sv.sendtoClient(msg, send_conn)
				sv.DoAbort(op.timestamp)
			} else {
				msg := strings.Join([]string{"OK", strconv.FormatInt(op.timestamp, 10)}, " ")
				sv.sendtoClient(msg, send_conn)
			}

		case "BALANCE":
			op := new(Operation)
			op.Init(operation)
			value, aborted := sv.read(*op)
			abort = aborted
			if abort {
				msg := strings.Join([]string{"NOT", strconv.FormatInt(op.timestamp, 10)}, " ")
				sv.sendtoClient(msg, send_conn)
				sv.DoAbort(op.timestamp)
			} else {
				msg := op.branch + "." + op.account + " = " + strconv.Itoa(value)
				msg = strings.Join([]string{msg, strconv.FormatInt(op.timestamp, 10)}, " ")
				sv.sendtoClient(msg, send_conn)
			}

		case "ABORT_Coordinator":
			timestamp, err := strconv.ParseInt(words[1], 10, 64)
			if err != nil {
				panic(err)
			}
			sv.DoAbort(timestamp)

		case "ABORTED", "NOT":
			text := ""
			if words[0] == "NOT" {
				text = "NOT FOUND, ABORTED"
			} else {
				text = words[0]
			}
			timestamp, err := strconv.ParseInt(words[1], 10, 64)
			if err != nil {
				panic(err)
			}
			sv.DoAbort(timestamp)
			msg := strings.Join([]string{"ABORT_Coordinator", strconv.FormatInt(timestamp, 10)}, " ")
			for _, name := range sv.name {
				if name != sv.me {
					send_conn := sv.send_conn[name]
					sv.sendtoClient(msg, send_conn)
				}
			}
			sv.sendtoClient(text, sv.send_conn[words[1]])
			sv.send_conn[strconv.FormatInt(timestamp, 10)].Close()
			sv.read_conn[strconv.FormatInt(timestamp, 10)].Close()

		case "COMMIT_PREPARE":
			timestamp, _ := strconv.ParseInt(words[1], 10, 64)
			abort = !sv.CheckCommit(timestamp)
			if abort {
				msg := strings.Join([]string{"ABORTED", strconv.FormatInt(timestamp, 10)}, " ")
				sv.sendtoClient(msg, send_conn)
				sv.DoAbort(timestamp)
			} else {
				msg := strings.Join([]string{"COMMIT_READY", strconv.FormatInt(timestamp, 10)}, " ")
				sv.sendtoClient(msg, send_conn)
			}

		case "COMMIT_READY":
			timestamp := words[1]
			sv.mu_commit.Lock()
			sv.commitCount[timestamp] += 1

			// if all other branches allow to commit the transaction
			if sv.commitCount[timestamp] == 4 {
				sv.mu_commit.Unlock()
				intstmp, err := strconv.ParseInt(words[1], 10, 64)
				if err != nil {
					panic(err)
				}
				sv.DoCommit(intstmp)
				msg := strings.Join([]string{"COMMIT_OK", timestamp}, " ")
				for _, name := range sv.name {
					if name != sv.me {
						sv.sendtoClient(msg, sv.send_conn[name])
					}
				}
				sv.sendtoClient("COMMIT OK", sv.send_conn[timestamp])
				sv.send_conn[timestamp].Close()
				sv.read_conn[timestamp].Close()
			} else {
				sv.mu_commit.Unlock()
			}

		case "COMMIT_OK":
			timestamp, err := strconv.ParseInt(words[1], 10, 64)
			if err != nil {
				panic(err)
			}
			sv.DoCommit(timestamp)

		default:
			text := strings.Join(words[:len(words)-1], " ")
			sv.sendtoClient(text, sv.send_conn[words[len(words)-1]])
		}
	}
}

func main() {
	argv := os.Args[1:]
	if len(argv) != ARG_NUM_SERVER {
		fmt.Fprintf(os.Stderr, "usage: ./server <Server Name [A,B,C,D,E]> <config.txt>\n")
		os.Exit(1)
	}
	sv := Server{
		mu_RTS:      &sync.Mutex{},
		mu_TW:       &sync.Mutex{},
		mu_commit:   &sync.Mutex{},
		me:          argv[0],
		address:     make(map[string]string),
		port:        make(map[string]string),
		name:        make(map[IP]string),
		accounts:    make(map[string]Account),
		send_conn:   make(map[string]net.Conn),
		read_conn:   make(map[string]net.Conn),
		TW:          make(map[string][]write_vale),
		RTS:         make(map[string][]int64),
		commitCount: make(map[string]int),
	}
	config_file := argv[1]

	sv.readFromConfig(config_file)
	sv.start_listen()
	sv.build_branches() // connect to all other branches

	for _, name := range sv.name {
		if name != sv.me {
			go sv.handleBranch(name, sv.read_conn[name], sv.send_conn[name])
		}
	}
	for {
		read_conn, send_conn := sv.connect_client() // connect once, always listen self' port
		go sv.handleConnection(read_conn, send_conn)
	}
}
