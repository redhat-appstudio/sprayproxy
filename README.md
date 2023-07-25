# Spray Proxy

A very simple reverse proxy that broadcasts to multiple backends.

## Try it

```sh
make build
bin/sprayproxy server --backend <backend-server> --backend <another-backend-server>
```

You can also configure the proxy with environment variables:

* `SPRAYPROXY_SERVER_HOST`: host for the proxy
* `SPRAYPROXY_SERVER_PORT`: port to serve the proxy
* `SPRAYPROXY_SERVER_BACKEND`: a space-separated list of backends to forward traffic. Example:

```sh
SPRAYPROXY_SERVER_BACKEND="http://localhost:8080 http://localhost:8081"
```

* `SPRAYPROXY_FORWARDING_REQUEST_TIMEOUT`: override the default forwarding request timeout. Default
  is 15 seconds.
* `SPRAYPROXY_MAX_REQUEST_SIZE`: override the default maximum request size. In bytes. Default is 25MB.
* `GH_APP_WEBHOOK_SECRET`: webhook secret for GitHub apps. See the
  [Github Apps guide](/docs/github-app.md) for more info.

The following environment variables are insecure and should not be used in production environments:

* `SPRAYPROXY_SERVER_INSECURE_SKIP_TLS_VERIFY`: Skip TLS verification when forwarding to backends.
* `SPRAYPROXY_SERVER_INSECURE_SKIP_WEBHOOK_VERIFY`: Skip GitHub webhook verification for incoming
  requests.
* `SPRAYPROXY_SERVER_ENABLE_DYNAMIC_BACKENDS`: Register and Unregister backends on the fly.
  **Note: this setting is for stateless deployment of the sprayproxy and should not be used in production and staging environments.**


## Developing

* Run `make build` to build the proxy sever (output to `bin/sprayproxy`)
* Run `make test` to run unit tests
* Run `make run` to launch the proxy with default configuration
