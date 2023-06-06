# Using Sprayproxy with GitHub Apps

Sprayproxy can be used to forward events from GitHub to backend servers that can process the event.
This document will show you how to configure Sprayproxy to forward these events.

## Prerequisites

* Deploy Sprayproxy with a publicly accessible endpoint.
* Create a [GitHub App](https://docs.github.com/en/apps/creating-github-apps/about-creating-github-apps)
  that is configured to forward the events of your choice (ex: pull requests, push events).

## GitHub App Configuration

* Set the [Webhook URL](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/using-webhooks-with-github-apps#choosing-a-webhook-url)
  to the sprayproxy's endpoint.
* Secure the webhook by setting a
  [webhook secret](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/using-webhooks-with-github-apps#securing-your-webhooks-with-a-webhook-secret).
* Record this secret's value in a secure location, such as [Vault](https://www.vaultproject.io/)
  or a cloud provider secret manager.

## Sprayproxy Configuration

* Create a secret named `gh-webhook-secret`, whose data should be they key/value pair
  `GH_APP_WEBHOOK_SECRET: <secret-value>`. Consider using a secured mechanism for syncing the
  webhook secret, such as the
  [External Secrets Operator](https://external-secrets.io/v0.8.3/).
* Set the `GH_APP_WEBHOOK_SECRET` environment variable in sprayproxy's deployment to match the
  webhook secret value above. This value should be stored in a Kubernetes secret that can be
  referenced using the [envFrom](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/#define-an-environment-variable-for-a-container)
  value option. Use following [Kustomize] patch as an example:

  ```yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: sprayproxy
    namespace: sprayproxy
  spec:
    containers:
      - name: sprayproxy
        envFrom:
          secretRef:
            name: gh-webhook-secret
  ```
