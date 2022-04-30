package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"crypto/rand"
	"math/big"
	"os"
	"strconv"
	"time"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

const Interval time.Duration = time.Second * 10

type MasterStatus int

const (
	Unassigned MasterStatus = iota
	InMap
	Intermediate
	InReduce
	Finished
)

const LengthOfChecksum int = 16

type TaskName int

type CoordinatorStatus struct {
	Done bool
}

const (
	Map TaskName = iota
	Reduce
)

var TaskNameMap = map[TaskName]string{
	Map:    "Map",
	Reduce: "Reduce",
}

type WorkerMeta struct {
	ReducerTotal int
}

type AskTaskParams struct {
	TaskCLS TaskName
}

type AskTaskReturn struct {
	WorkerId int
	FileName []string
	Checksum string
}

type ReportDoneParams struct {
	WorkerId int // reduce,map use this to identify itself
	Checksum string
	TaskCLS  TaskName
}

type ReportDoneReturn struct {
	Got bool
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}

func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}
