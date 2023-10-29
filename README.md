# Livekit Event Brige

Event bridge for livekit webhook events. Forward webhook events to websocket or redis pub/sub.

## Configure

Minimal setup 
```yaml
...
livekit:
    api_key: LIVEKIT_API_KEY
    api_secret: LIVEKIT_API_SECRET


... (optional)
server:
    port: default 3000
    bind_address: default 0.0.0.0

log_level: default debug

# connect priority if set
# sentinel -> cluster -> single
redis: 
    address: host:port
    username:
    password:
    db:
    channel: default (livekit-events)

    #timeouts
    dial_timeout:
    read_timeout:
    write_timeout:
    
    # sentinels
    sentinel_master:
    sentinel_username:
    sentinel_password:
    sentinel_addresses:

    # cluster
    cluster_addresses:
    cluster_max_redirects:


```


## Contribute

Setup

```bash
make all
```

This will initialize a git repo, download the dependencies in the latest versions and install all needed tools.
If needed code generation will be triggered in this target as well.

## Test & lint

Run linting

```bash
make lint
```

Run tests

```bash
make test
```
