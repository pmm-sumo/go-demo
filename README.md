# instrumentation example

The example consists of two parts:

* An HTTP server using gorilla/mux and instrumentation. The server has a
`/users/{id:[0-9]+}` endpoint. 
* A client that uses net/http

