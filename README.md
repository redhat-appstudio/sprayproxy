# Spray Proxy

A very simple reverse proxy that broadcasts to multiple backends.

## Try it!

```sh
$ make build
$ bin/sprayproxy server --backend <backend-server> --backend <another-backend-server>
```

You can also configure the proxy with environment variables:

* `SPRAYPROXY_SERVER_HOST`: host for the proxy
* `SPRAYPROXY_SERVER_PORT`: port to serve the proxy
* `SPRAYPROXY_SERVER_BACKENDS`: a space-separated list of backends to forward traffic. Example:

  ```
  SPRAYPROXY_SERVER_BACKENDS="http://localhost:8080 http://localhost:8081"
  ```

## Developing

* Run `make build` to build the proxy sever (output to `bin/sprayproxy`)
* Run `make test` to run unit tests
* Run `make run` to launch the proxy with default configuration
