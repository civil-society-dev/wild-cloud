# External DNS

See: https://github.com/kubernetes-sigs/external-dns

ExternalDNS allows you to keep selected zones (via --domain-filter) synchronized with Ingresses and Services of type=LoadBalancer and nodes in various DNS providers.

Currently, we are only configured to use CloudFlare.

Docs: https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/cloudflare.md

Any Ingress that has metatdata.annotions with
external-dns.alpha.kubernetes.io/hostname: `<something>.${DOMAIN}`

will have Cloudflare records created by External DNS.
