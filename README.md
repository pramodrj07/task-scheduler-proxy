# Task Scheduler proxy with Rate limiting

# Instructions

###  1. Install golang
###  2. `cd task-scheduler-proxy`
###  3. `go mod tidy`
###  4. `go run main.go`

###  5. Open your terminal and run the following command to test the API:
`seq 1 10 | xargs -n1 -P10 -I{} curl -s http://localhost:8080/process
xargs: warning: options --max-args and --replace/-I/-i are mutually exclusive, ignoring previous --max-args value`

###  6. Observe that only 5 requests are processed at a time, and the rest are rejected.
