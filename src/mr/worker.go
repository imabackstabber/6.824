package mr

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"time"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

var nReduce int

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// need to pay attention to output format
	// just like "%v %v"

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()
	Init()
	// Map stage
	for {
		if TestFinish(Map) {
			break
		} else {
			err := AskTask(Map, mapf, reducef)
			if err != nil {
				time.Sleep(time.Second * 2)
			}
		}
	}
	// Reduce stage
	for {
		if TestFinish(Reduce) {
			break
		} else {
			err := AskTask(Reduce, mapf, reducef)
			if err != nil {
				time.Sleep(time.Second * 2)
			}
		}
	}
	fmt.Printf("worker leaving now\n")
}

func AskTask(tn TaskName, mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) error {
	args := AskTaskParams{}
	args.TaskCLS = tn
	reply := AskTaskReturn{}
	ok := call("Coordinator.Dispatch", &args, &reply)
	if ok {
		fmt.Printf("%v workerID:%v got %v task:%v,checksum:%v\n", time.Now().Format(time.ANSIC), reply.WorkerId, TaskNameMap[args.TaskCLS], reply.FileName, reply.Checksum)
	} else {
		return errors.New("ask for task but didn't get it")
	}
	if tn == Map {
		reportargs := ReportDoneParams{reply.WorkerId, reply.Checksum, Map}
		DoMapTask(&reportargs, reply.FileName, mapf)
	} else {
		reportargs := ReportDoneParams{reply.WorkerId, reply.Checksum, Reduce}
		DoReduceTask(&reportargs, reply.FileName, reducef)
	}
	return nil
}

func DoMapTask(args *ReportDoneParams, filenames []string, mapf func(string, string) []KeyValue) error {
	workid := args.WorkerId
	fname := filenames[0] // for map,only one split
	file, err := os.Open(fname)
	if err != nil {
		log.Fatalf("cannot open %v", fname)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", fname)
	}
	file.Close()
	kva := mapf(fname, string(content)) // []KeyValue
	fnamepre := "mr-" + strconv.Itoa(workid)
	filekva := make([][]KeyValue, nReduce) // nReduce
	for _, kv := range kva {
		key, _ := kv.Key, kv.Value
		reduceBucket := ihash(key) % nReduce
		filekva[reduceBucket] = append(filekva[reduceBucket], kv)
	}
	for i, kva := range filekva {
		interfname := fnamepre + "-" + strconv.Itoa(i)
		file, err := ioutil.TempFile("", "temp-")
		if err != nil {
			log.Fatalf("cannot create %v", interfname)
		}
		enc := json.NewEncoder(file)
		for _, kv := range kva {
			enc.Encode(&kv)
		}
		// enc.Encode(&kva)
		os.Rename(file.Name(), interfname)
		file.Close()
	}
	reply := ReportDoneReturn{}
	ok := call("Coordinator.ReportDone", &args, &reply)
	if !ok {
		fmt.Printf("worker %v Cannot call\n", workid)
	}
	if reply.Got {
		fmt.Printf("%v worker %v finish map,exiting\n", time.Now().Format(time.ANSIC), workid)
	} else {
		fmt.Printf("map task has been reissued\n")
	}
	return nil
}

func DoReduceTask(args *ReportDoneParams, fnames []string, reducef func(string, []string) string) error {
	workid := args.WorkerId

	// build a mapper
	mapper := make(map[string][]string)
	for _, fname := range fnames {
		file, err := os.Open(fname)
		if err != nil {
			log.Fatalf("cannot open %v", fname)
		}
		dec := json.NewDecoder(file)
		kva := make([]KeyValue, 0)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			kva = append(kva, kv)
		}
		file.Close()
		sort.Sort(ByKey(kva))

		//
		// call Reduce on each distinct key in intermediate[],
		// and print the result to mr-out-0.
		//
		i := 0
		for i < len(kva) {
			j := i + 1
			for j < len(kva) && kva[j].Key == kva[i].Key {
				j++
			}
			for k := i; k < j; k++ {
				mapper[kva[k].Key] = append(mapper[kva[k].Key], kva[k].Value)
			}
			i = j
		}
	}

	oname := "mr-out-" + strconv.Itoa(workid)
	ofile, err := ioutil.TempFile("", "temp-")
	if err != nil {
		panic(err)
	}
	keys := make([]string, 0)
	for k, _ := range mapper {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(ofile, "%v %v\n", k, reducef(k, mapper[k]))
	}
	os.Rename(ofile.Name(), oname)
	ofile.Close()

	reply := ReportDoneReturn{}
	ok := call("Coordinator.ReportDone", &args, &reply)
	if !ok {
		fmt.Printf("worker %v Cannot call\n", workid)
	}
	if reply.Got {
		fmt.Printf("%v worker %v finish reduce,exiting\n", time.Now().Format(time.ANSIC), workid)
	} else {
		fmt.Printf("reduce task has been reissued\n")
	}
	return nil
}

func TestFinish(tn TaskName) bool {
	args := tn
	reply := CoordinatorStatus{}
	ok := call("Coordinator.IsStageDone", &args, &reply)
	if !ok {
		fmt.Printf("worker cannot call coordinator,just exit worker\n")
		return true
	}
	return reply.Done
}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//

func Init() {
	// declare an argument structure.
	args := struct{}{}

	// declare a reply structure.
	reply := WorkerMeta{}

	ok := call("Coordinator.GetMeta", &args, &reply)
	if ok {
		fmt.Printf("Get WorkerMeta, Reducer num:%v\n", reply.ReducerTotal)
	} else {
		log.Fatal("worker init call failed!\n")
	}
	nReduce = reply.ReducerTotal
}

func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname) // it use a local unix file system
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
