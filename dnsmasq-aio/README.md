# dnsmasq cluster

A simple cluster running on:
- native Docker containers
- kubernetes

With [dnsmasq-china-list](https://github.com/felixonmars/dnsmasq-china-list) and a clean & reliable DNS upstream.

> You need a IEPL / IPLC connection or other way (e.g. [DoH](https://developers.cloudflare.com/1.1.1.1/dns-over-https/cloudflared-proxy/) / DoT) query DNS server to avoid ISP's DNS pollution.

The default DNS upstream is `8.8.4.4` which has a node located at Hong Kong's Google anycast network. Domains which is hosted at China mainland was load from [dnsmasq-china-list](https://github.com/felixonmars/dnsmasq-china-list), Chinese default DNS upstream is `114.114.114.114`, other famous public DNS server like `119.29.29.29`, `223.5.5.5` are also good.

### How to use

You need [Docker](https://www.docker.com/get-started) and [Docker Compose](https://docs.docker.com/compose/) or [kubernetes](https://kubernetes.io/docs/tutorials/kubernetes-basics/) installed.

#### Run with Docker Compose



#### Run with Kubernetes

