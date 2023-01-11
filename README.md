# Spray Proxy

A very simple reverse proxy that broadcasts to multiple backends.

## Try it!

```sh
$ make build
$ bin/sprayproxy server --backend <backend-server> --backend <another-backend-server>
```

## Developing

* Run `make build` to build the proxy sever (output to `bin/sprayproxy`)
* Run `make test` to run unit tests
* Run `make run` to launch the proxy with default configuration
