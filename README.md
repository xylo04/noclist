# NOC List

This is my solution to the
[Ad Hoc Homework](https://homework.adhoc.team/) [NOC List](https://homework.adhoc.team/noclist/)
assignment. It is an HTTP client which fetches resources from an HTTP API. The API requires an
interesting authentication technique and assumed to not be very reliable.

Compiling this project requires a Go v1.17+ development environment.

Unit tests can be run by:

```shell
make test
```

The binary can be compiled with:

```shell
make
```

As described in the assignment, a server implementing the API can be run with `docker`:

```shell
docker run --rm -p 8888:8888 adhocteam/noclist
```

Once the server is running, this project's client can be run by:

```shell
./noclist
```

This client outputs JSON to STDOUT, and informational logs are only sent to STDERR. This means the
client can be chained with tools like `jq`:

```shell
./noclist | jq .
```
