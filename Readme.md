# Toonels

Open tunnels with ease.

## Example

You'll need a file `.tunnels.yaml`:

```yaml
---
nodes:
  - user: toon
    private_key_path: /home/toon/.ssh/id_rsa
    addr: 10.0.0.123:22
    tunnels:
      - local: 127.0.0.1:6443
        target: 10.0.0.122:6443
      - local: 127.0.0.1:8080
        target: 10.0.0.133:80
```
