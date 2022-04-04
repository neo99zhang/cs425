package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, Term, isleader)
//   start agreement on a new log entry
// rf.GetState() (Term, isLeader)
//   ask a Raft for its current Term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"../labrpc"
)

//
// as each Raft peer becomes aware that Successive log Entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

// Some extra structs needed defined here
type LogEntry struct {
	Term    int
	Command interface{}
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu    sync.Mutex          // Lock to protect shared access to this peer's state
	peers []*labrpc.ClientEnd // RPC end points of all peers
	me    int                 // this peer's index into peers[]
	dead  int32               // set by Kill()

	// Your data here (2A, 2B).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	// You may also need to add other state, as per your implementation.

	// Persistent state on all servers
	//Start: {0:Leader,1:Candidate,2:Follower}
	state       int
	currentTerm int
	votedFor    int
	log         []LogEntry
	//TODO some timer for timeout?

	// Volatile state on all servers:
	commitIndex int
	lastApplied int

	// Volatile state on leaders:
	nextIndex  []int
	matchIndex []int

	//extra
	applyMsgCh     chan ApplyMsg
	electionTimer  *time.Timer
	heartBeatTimer *time.Timer
}

const (
	LEADER              = 0
	CANDIDATE           = 1
	FOLLOWER            = 2
	NULL                = -1
	ELECTIONTIMEOUT_MAX = 500
	ELECTIONTIMEOUT_MIN = 150
	HEARTBEAT_INTERVAL  = 110 * time.Millisecond
)

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var Term int
	var isleader bool
	// Your code here (2A).

	rf.mu.Lock()
	Term = rf.currentTerm
	isleader = rf.state == LEADER
	rf.mu.Unlock()
	return Term, isleader
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) TurnFollower(Term int, voteFor int) {
	if rf.state == FOLLOWER {
		// fmt.Print("Server: ", rf.me, " changes term from ", rf.currentTerm, " to ", Term, "\n")
		rf.currentTerm = Term
		rf.votedFor = voteFor

		return
	} else {
		// fmt.Print("Server: ", rf.me, " becomes follower at term: ", Term, "\n")
	}

	rf.state = FOLLOWER
	rf.currentTerm = Term
	rf.votedFor = voteFor
	rf.heartBeatTimer.Stop()
	//rf.electionTimer.Reset(randomTimeoutVal(ELECTIONTIMEOUT_MIN, ELECTIONTIMEOUT_MAX))
}

func (rf *Raft) TurnCandidate() {
	rf.state = CANDIDATE
	// fmt.Print("Server: ", rf.me, " becomes candidate at term: ", rf.currentTerm, "\n")
	rf.currentTerm += 1
	rf.votedFor = rf.me
	rf.heartBeatTimer.Stop()
	rf.electionTimer.Reset(randomTimeoutVal(ELECTIONTIMEOUT_MIN, ELECTIONTIMEOUT_MAX))
}

func (rf *Raft) TurnLeader() {
	rf.mu.Lock()
	rf.state = LEADER
	// fmt.Print("Server: ", rf.me, " becomes Leader at term: ", rf.currentTerm, "\n")
	rf.electionTimer.Stop()

	for i := range rf.peers {
		rf.nextIndex[i] = len(rf.log)
		rf.matchIndex[i] = 0
	}
	rf.mu.Unlock()
	for i := range rf.peers {
		if i != rf.me {
			// send the heartbeat
			go rf.sendHeartBeat(i)
		}
	}
	rf.heartBeatTimer.Reset(HEARTBEAT_INTERVAL)
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	// Read the fields in "args",
	// and accordingly assign the values for fields in "reply".
	if rf.killed() {
		return
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()
	// 1. Reply false if Term < currentTerm (§5.1)
	if (args.Term < rf.currentTerm) || (args.Term == rf.currentTerm && rf.votedFor != NULL && rf.votedFor != args.CandidateId) {
		reply.VoteGranted = false
		reply.Term = rf.currentTerm
		return
	}
	if args.Term > rf.currentTerm {
		//set as follower
		rf.TurnFollower(args.Term, NULL)
	} //check
	// 2. If votedFor is null or CandidateId, and candidate’s log is at
	// least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
	reply.Term = args.Term
	if rf.votedFor == NULL || rf.votedFor == args.CandidateId {
		if len(rf.log) == 0 || rf.log[len(rf.log)-1].Term < args.LastLogTerm ||
			(rf.log[len(rf.log)-1].Term == args.LastLogTerm && len(rf.log)-1 <= args.LastLogIndex) {
			rf.votedFor = args.CandidateId
			reply.VoteGranted = true
			rf.electionTimer.Reset(randomTimeoutVal(ELECTIONTIMEOUT_MIN, ELECTIONTIMEOUT_MAX))
			return
		}
	}
	reply.VoteGranted = false
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// 1. Return if Term < currentTerm
	if rf.killed() {
		return
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	// 2. If Term > currentTerm, currentTerm ← Term
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.TurnFollower(args.Term, NULL)
	}

	// 3. If candidate or leader, step down

	// 4. Reset election timeout
	rf.electionTimer.Reset(randomTimeoutVal(ELECTIONTIMEOUT_MIN, ELECTIONTIMEOUT_MAX))
	// 5. Return failure if log doesn’t contain an entry at
	// PrevLogIndex whose Term matches PrevLogTerm'
	// fmt.Printf("Server: %v len(rf.log)-1: %v, args.PrevLogIndex: %v \n", rf.me, len(rf.log)-1, args.PrevLogIndex)
	// if len(rf.log)-1 >= args.PrevLogIndex {
	// 	fmt.Printf("Server: %v rf.log[(args.PrevLogIndex)].Term: %v , args.PrevLogTerm: %v\n", rf.me, rf.log[(args.PrevLogIndex)].Term, args.PrevLogTerm)
	// }
	if (len(rf.log)-1 < args.PrevLogIndex) || (rf.log[(args.PrevLogIndex)].Term != args.PrevLogTerm) {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	// 6. If existing Entries conflict with new Entries, delete all
	// existing Entries starting with first conflicting entry
	point := NULL
	for i, entry := range args.Entries {
		curr_idx := args.PrevLogIndex + 1 + i
		if (curr_idx > len(rf.log)-1) || (rf.log[curr_idx].Term != entry.Term) {
			point = curr_idx
			break
		}
	}
	// 7. Append any new Entries not already in the log
	if point != NULL {
		rf.log = append(rf.log[:point], args.Entries[point-1-args.PrevLogIndex:]...)
	}
	//  else {
	// 	fmt.Printf("Server: %v Point is NULL, PrevLogIndex: %v, PrevLogTerm: %v,  self log length: %v \n", rf.me, args.PrevLogIndex, args.PrevLogTerm, len(rf.log))
	// }
	// 8. Advance state machine with newly committed Entries

	if args.LeaderCommit > rf.commitIndex {
		// fmt.Print("Server: ", rf.me, " match at point: ", point, "\n")
		// fmt.Print("Server: ", rf.me, " commitIndex change from ", rf.commitIndex, " to ", args.LeaderCommit, "\n")
		rf.commitIndex = Min(len(rf.log), args.LeaderCommit)
	}

	rf.applymsg()
	reply.Success = true
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// Term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {

	index := -1
	Term := -1
	isLeader := true

	// Your code here (2B).
	rf.mu.Lock()
	isLeader = rf.state == LEADER
	Term = rf.currentTerm

	// return false if it is not the leader
	if !isLeader {
		rf.mu.Unlock()
		return -1, -1, false
	}

	// append the logentry to log
	new_entry := LogEntry{
		Term:    rf.currentTerm,
		Command: command,
	}
	rf.log = append(rf.log, new_entry)
	rf.matchIndex[rf.me] = len(rf.log) - 1
	rf.nextIndex[rf.me] = len(rf.log)
	index = len(rf.log) - 1
	rf.mu.Unlock()

	return index, Term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

/*
 * Helper function for Make()
 */

func (rf *Raft) maintain() {
	//timeout,enter cs,change status

	for !rf.killed() {
		rf.mu.Lock()
		isLeader := rf.state == LEADER
		rf.mu.Unlock()

		if isLeader {
			rf.handleLeader()
		} else {
			rf.checkElectionTimeout()
		}

		if rf.killed() {
			// fmt.Print("Found ", rf.me, " just killed\n")
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return
}

func (rf *Raft) handleLeader() {
	if rf.killed() {
		return
	}
	rf.mu.Lock()
	peers := rf.peers
	me := rf.me
	rf.mu.Unlock()
	select {
	case <-rf.heartBeatTimer.C:
		for i := range peers {
			if i != me {
				// send the heartbeat
				go rf.sendHeartBeat(i)
			}
		}
		rf.heartBeatTimer.Reset(HEARTBEAT_INTERVAL)
	}

}

func (rf *Raft) sendHeartBeat(peerId int) {
	if rf.killed() {
		return
	}
	rf.mu.Lock()
	entries := make([]LogEntry, len(rf.log[(rf.nextIndex[peerId]):]))
	copy(entries, rf.log[(rf.nextIndex[peerId]):])
	args := AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: rf.nextIndex[peerId] - 1,
		PrevLogTerm:  rf.log[rf.nextIndex[peerId]-1].Term,
		Entries:      entries,
		LeaderCommit: rf.commitIndex,
	}
	rf.mu.Unlock()
	reply := AppendEntriesReply{}

	Success := rf.sendAppendEntries(peerId, &args, &reply)
	if rf.killed() {
		return
	}
	if !Success {
		// fmt.Print("Leader: ", rf.me, "with term", rf.currentTerm, " cannot send entry to ", peerId, "\n")
		return
	}
	// fmt.Print("Leader: ", rf.me, "with term", rf.currentTerm, " successfully sent entry to ", peerId, "\n")

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.state != LEADER {
		return
	}

	// check Term
	if reply.Term > rf.currentTerm {
		rf.TurnFollower(reply.Term, NULL)
		return
	}

	// if reply succeeds
	if reply.Success {
		rf.nextIndex[peerId] = len(rf.log)
		rf.matchIndex[peerId] = rf.nextIndex[peerId] - 1
		for i := len(rf.log) - 1; i > rf.commitIndex; i-- {
			total := 0
			for j := range rf.matchIndex {
				if rf.matchIndex[j] >= i {
					total += 1
				}
			}
			half := len(rf.peers) / 2
			if total > half {
				// fmt.Print("Leader: ", rf.me, " commitIndex change from ", rf.commitIndex, " to ", i)
				rf.commitIndex = Max(i, rf.commitIndex)
				// fmt.Print(" with the last commited log's term is: ", rf.log[rf.commitIndex].Term, "\n")
				break
			}
		}
		rf.applymsg()

		return
	} else {
		// fmt.Print("Leader: ", rf.me, " sends heart beat with nextIndex ", rf.nextIndex[peerId], " to ", peerId, " fail\n")
		rf.nextIndex[peerId]--
	}
}

func (rf *Raft) startElection() {
	//start election
	if rf.killed() {
		return
	}
	rf.mu.Lock()
	args := RequestVoteArgs{
		// Your data here (2A, 2B).
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: len(rf.log) - 1,
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
	}
	rf.mu.Unlock()
	var voteNum int32 = 1
	for i := range rf.peers {
		if i != rf.me {
			// send the heartbeat
			go rf.sendvote(i, &args, &voteNum)
		}
	}

}

func (rf *Raft) sendvote(peerId int, args *RequestVoteArgs, voteNum *int32) {
	if rf.killed() {
		return
	}
	reply := RequestVoteReply{}
	Success := rf.sendRequestVote(peerId, args, &reply)
	if rf.killed() {
		return
	}
	if !Success {
		// fmt.Print("Candidate: ", rf.me, "cannot send requestvote to ", peerId, "\n")
		return
	}
	// step down if get a larger Term
	if reply.Term > rf.currentTerm {
		rf.mu.Lock()
		rf.TurnFollower(reply.Term, NULL)
		rf.mu.Unlock()
	}
	rf.mu.Lock()
	if reply.VoteGranted && (rf.state == CANDIDATE) {
		atomic.AddInt32(voteNum, 1)
		if atomic.LoadInt32(voteNum) > int32(len(rf.peers)/2) {
			rf.state = LEADER
			rf.mu.Unlock()
			rf.TurnLeader()
			return
		}
	}
	rf.mu.Unlock()

}

func (rf *Raft) checkElectionTimeout() {
	if rf.killed() {
		return
	}
	rf.mu.Lock()
	select {
	case <-rf.electionTimer.C:
		// fmt.Print("Server: ", rf.me, " Timeout\n")
		rf.TurnCandidate()
		rf.mu.Unlock()
		rf.startElection()
	default:
		rf.mu.Unlock()
	}

}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.me = me
	rf.applyMsgCh = applyCh
	// Your initialization code here (2A, 2B).

	rf.log = make([]LogEntry, 1)
	rf.state = FOLLOWER
	rf.currentTerm = 0
	rf.votedFor = NULL
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))
	for i := range rf.nextIndex {
		rf.nextIndex[i] = len(rf.log)
	}
	//how to use:
	//reset: rf.electionTimer.Reset(randomTimeoutVal(ELECTIONTIMEOUT_MIN,ELECTIONTIMEOUT_MAX))
	//
	rf.electionTimer = time.NewTimer(randomTimeoutVal(ELECTIONTIMEOUT_MIN, ELECTIONTIMEOUT_MAX))
	rf.heartBeatTimer = time.NewTimer(HEARTBEAT_INTERVAL)

	go rf.maintain()

	return rf
}

//get a randomized time out value in milliseconds
func randomTimeoutVal(min int, max int) time.Duration {
	return time.Duration(min+rand.Intn(max-min)) * time.Millisecond
}

func (rf *Raft) applymsg() {
	if rf.killed() {
		return
	}
	for n := rf.commitIndex - rf.lastApplied; n > 0; n-- {
		msg := ApplyMsg{
			CommandValid: true,
			Command:      rf.log[rf.commitIndex+1-n].Command,
			CommandIndex: rf.commitIndex + 1 - n,
		}

		rf.applyMsgCh <- msg
		if rf.state == LEADER {
			// fmt.Print("Leader: ", rf.me, " apply msg with Index ", msg.CommandIndex, "\n")
		} else {
			// fmt.Print("Follower: ", rf.me, " apply msg with Index ", msg.CommandIndex, "\n")
		}

	}
	// rf.mu.Lock()
	rf.lastApplied = rf.commitIndex
	// rf.mu.Unlock()
}
