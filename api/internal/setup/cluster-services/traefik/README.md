# Traefik

- https://doc.traefik.io/traefik/providers/kubernetes-ingress/

Ingress RDs can be create for any service. The routes specificed in the Ingress are added automatically to the Traefik proxy.

Traefik serves all incoming network traffic on ports 80 and 443 to their appropriate services based on the route.

## Notes

These kustomize templates were created with:

```bash
helm-chart-to-kustomize traefik/traefik traefik traefik values.yaml
```

With values.yaml being:

```yaml
ingressRoute:
  dashboard:
    enabled: true
    matchRule: Host(`dashboard.localhost`)
    entryPoints:
      - web
providers:
  kubernetesGateway:
    enabled: true
gateway:
  namespacePolicy: All
```
