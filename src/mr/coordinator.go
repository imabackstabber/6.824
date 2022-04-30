package mr

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

type Coordinator struct {
	// Your definitions here.
	// Add lock to dispatch
	sync.Mutex
	files          []string
	filesmap       map[string]MasterStatus
	mapchecksum    map[string]string
	count          int
	mapfinished    int
	reducefinished int
	ongoingreduce  map[int]MasterStatus
	reducechecksum map[int]string
	nReduce        int
}

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

func (c *Coordinator) GetMeta(args *struct{}, reply *WorkerMeta) error {
	reply.ReducerTotal = c.nReduce
	return nil
}

func (c *Coordinator) Dispatch(args *AskTaskParams, reply *AskTaskReturn) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	// generate checksum to check correnspondence of task and worker
	checksum, e := GenerateRandomString(LengthOfChecksum)
	if e != nil {
		return errors.New("can't generate checksum ")
	} else {
		reply.Checksum = checksum
	}
	reply.FileName = make([]string, 0)

	if args.TaskCLS == Map {
		reply.WorkerId = 0 // as a counter

		for _, key := range c.files {
			value := c.filesmap[key]
			if value == Unassigned {
				// reply.FileName = key
				reply.FileName = append(reply.FileName, key)
				c.filesmap[key] = InMap
				c.mapchecksum[key] = reply.Checksum
				break
			}
			reply.WorkerId += 1
		}
		if reply.WorkerId == c.count {
			return errors.New("can't assign map task,just idle")
		} else {
			go func(mapfilename string) {
				// detect abnormal worker
				time.Sleep(Interval)
				c.Mutex.Lock()
				defer c.Mutex.Unlock()
				if c.filesmap[mapfilename] != Intermediate {
					// allow to relaunch
					c.filesmap[mapfilename] = Unassigned
					c.mapchecksum[mapfilename] = ""
				}
			}(reply.FileName[0])
		}
	} else {
		reply.WorkerId = -1
		for k, v := range c.ongoingreduce {
			if v == Intermediate {
				reply.WorkerId = k
				for i := 0; i < c.count; i++ {
					reply.FileName = append(reply.FileName, "mr-"+strconv.Itoa(i)+"-"+strconv.Itoa(reply.WorkerId))
				}
				c.ongoingreduce[k] = InReduce
				c.reducechecksum[k] = reply.Checksum
				break
			}
		}
		if reply.WorkerId == -1 {
			return errors.New("can't assign Reduce task,just idle")
		} else {
			go func(reduceworkerid int) {
				// detect abnormal worker
				time.Sleep(Interval)
				c.Mutex.Lock()
				defer c.Mutex.Unlock()
				if c.ongoingreduce[reduceworkerid] != Finished {
					c.ongoingreduce[reduceworkerid] = Intermediate
				}
			}(reply.WorkerId)
		}
	}
	return nil
}

func (c *Coordinator) ReportDone(args *ReportDoneParams, reply *ReportDoneReturn) error {
	workerid, taskCLS, checksum := args.WorkerId, args.TaskCLS, args.Checksum
	// should test if it is still valid
	// may dispatch file to a new worker
	// just ignore it here,for now

	// better set a single lock for each file
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if taskCLS == Map {
		fname := c.files[args.WorkerId]
		if (c.filesmap[fname] == InMap) && (c.mapchecksum[fname] == checksum) {
			c.filesmap[fname] = Intermediate
			c.mapfinished += 1
			reply.Got = true
		} else {
			reply.Got = false
			return errors.New("map task has been reissued")
		}
	} else {

		if (c.ongoingreduce[workerid] == InReduce) && (c.reducechecksum[workerid] == checksum) {
			c.ongoingreduce[workerid] = Finished
			c.reducefinished += 1
			reply.Got = true
		} else {
			reply.Got = false
			return errors.New("reduce task has been reissued")
		}
	}
	return nil
}

func (c *Coordinator) IsStageDone(args *TaskName, reply *CoordinatorStatus) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if *args == Map {
		if c.mapfinished == c.count {
			fmt.Printf("%v coordinator finished map\n", time.Now().Format(time.ANSIC))
			reply.Done = true
		} else {
			reply.Done = false
		}
	} else {
		if c.reducefinished == c.nReduce {
			fmt.Printf("%v coordinator finished reduce\n", time.Now().Format(time.ANSIC))
			reply.Done = true
		} else {
			reply.Done = false
		}
	}
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if c.reducefinished == c.nReduce {
		ret = true
	}
	return ret
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	// assign
	c.files = make([]string, 0)
	c.filesmap = make(map[string]MasterStatus)
	for _, file := range files {
		c.files = append(c.files, file)
		c.filesmap[file] = Unassigned
	}
	c.count = len(files)
	c.nReduce = nReduce
	c.ongoingreduce = make(map[int]MasterStatus)
	for i := 0; i < c.nReduce; i++ {
		c.ongoingreduce[i] = Intermediate
	}
	c.mapchecksum = make(map[string]string)
	c.reducechecksum = make(map[int]string)

	fmt.Printf("%v coordinator launching\n", time.Now().Format(time.ANSIC))

	c.server()
	return &c
}
