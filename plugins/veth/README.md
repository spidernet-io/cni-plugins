# veth-plugin

## Why

## How to start

### Examples

```shell
{
    "cniVersion": "0.3.0",
    "name": "macvlan-veth",
    "plugins": [
        {
            "type": "macvlan",
            "master": "enp4s0f1np1",
            "mode": "bridge",
            "ipam": {
                "default_ipv4_ippool": [
                    "default-v4-ippool"
                ],
                "type": "spiderpool",
                "log_level": "DEBUG",
                "log_file_path": "/var/log/spidernet/spiderpool.log",
                "log_file_max_size": 100,
                "log_file_max_age": 30,
                "log_file_max_count": 10
            }
        },{
            "type": "veth",
            "routes": [
                {
                    "dst": "10.240.0.0/12"
                },{
                    "dst": "172.96.0.0/18"
                }
            ],
            "rp_filter": {
                "enable": true,
                "value": 2
            },
            "skip": false
        }
    ]
}
```

## Notice 
