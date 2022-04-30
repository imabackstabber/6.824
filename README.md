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