# simple-dns

It serves DNS records from a local file (`zone.txt`) and forwards all other requests to an upstream server like Google DNS.

Create a `zone.txt` file:
This file holds your local DNS records.
```
myserver.local.   IN  A   192.168.1.10
nas.local.        IN  A   192.168.1.50
```

Run the server:
use the standard DNS port 53.
```bash
sudo ./simsam_dns
```

can change the default settings with flags:

*   `-port <number>`: Set the listening port (default: `53`).
*   `-zone_file <path>`: Set the path to your zone file (default: `./zone.txt`).
*   `-server <ip:port>`: Set the upstream DNS server (default: `8.8.8.8:53`).
    *   To disable forwarding: `sudo ./simsam_dns -server ""`
