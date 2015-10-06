# Very simple log router
```
printf "[example1-key] some log message\r\n[example2-key] another message\r\n" | go run lgroute.go "[example1-key]>./example1.log" "[example2-key]>>./example2.log"
```
In this sample, line with key [example1-key] piped to file example1.log with truncate file before write, and line with key [example2-key] appends to file example2.log
 
That's it's :)