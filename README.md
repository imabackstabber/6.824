## 6.824

## Lab 0:No Plagiarism

This repo is for those who take 6.824 as a self-teaching course to take as reference.

## Lab 1:MapReduce

0. There's no need to make multiple worker(e.g.,`go AskTask(...)`) in `worker.go`.In this task, we use multiple terminal executing `worker.go` on the same machine.

1. Pay attention that Go RPC sends only struct fields whose names start with capital letters.Sub-structures must also have capitalized field names.
2. Use lock to protect your shared data,e.g. Map(map in go)
3. When you iterating Map in golang, it does not guarantee same order is returned every time. So use a slice to store key and sort slice to make sure the order is correct
4. For crash test, just use a goroutine to make sure that jobs can be re-issued.Namely,just `sleep(interval)`,and check whether it's done or not, if noter-issue
5. I use a checksum to make sure the jobs wouldn't be repeatedly reported to be done.e.g. ,task has been re-issued, but the original worker still report the task(probably due to slow execution ),we just abandon its report by checking the checksum worker reported and  coordinator keeps.

It has passed test-mr-many.sh,using `numTrials = 10`

P.S.:Each file corresponds to one "split", and is the input to one Map task. However,for REAL MAPREDUCE,it will **split** each file and then dispatch it.

P.P.S.:We shall get `NumOfReduce`(in this lab,10) reduce output files, named `mr-out-X`

## Lab 2:Raft

### Part 2-A

0. It's important to sketch a layout for raft. Because raft basically follows the `one strong leader` pattern, the state of each individual server become important: There shall be only one leader and one can become a leader iff it has received more than half number of vote.

1. Concurrency model. Because of using Golang, Raft basically need a main thread(goroutine) to detect the state transition of servers. Apart from that, it has multiple sub goroutine running at the same time.(e.g. RequestVote as a candidate, SendHeartbeat as a leader). It's crucial  that one implement this concurrency model correctly, to make sure message can pass in time to maintain the correctness of the    raft system.

   <img src="/Users/alex/git_repo/6.824/asset/raft-图4.png" alt="图 4 " style="zoom:50%;" />

> Fig2-a.1 Copied from Fig.4 in `Rust-extended`.

2. It's important to know when to use `go <func>` to launch a goroutine, and processing return value of goroutine by using **channels**. For example, one candidate need to ask for vote to everyone, then there's no need to use a for loop and wait for return value and process it. Instead, use `go <func>` and send return value to channel

3. Use the fucking `-race` flag. If you don't know how to run a specific test with a race flag, here is a example shows how to run 2A with race flag:

   ```bash
   go test -run 2A -race
   ```

   That will make your life easier.

4. Be careful when using lock. Since go will not prevent you using lock to do anything you want (In rust, complier freaks out, sometimes it recycle the lock you use in callee function automatically), pay attention how to use lock(e.g. you might accidentally leave function before unlock the lock) and when to use it(e.g. coarse/fine granularity consideration)