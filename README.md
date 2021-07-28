# curtail
Achieve `tail -f` feature over http using `curl` command.

## Usage

Execute the `curtail` command on the server side:

```bash
$ curtail -addr 127.0.0.1:8080 /var/log/syslog
```

On the client side, follow the syslog messages using `curl` command:

```bash
$ curl -N http://127.0.0.1:8080

# You can also filter messages using query-strings:
$ curl -N http://127.0.0.1:8080?q=auth

# This is equivalent to:
$ curl -N http://127.0.0.1:8080 | grep --line-buffered auth
```
