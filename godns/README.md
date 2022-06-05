## Usage
First of all, you should clone this repo with:
```sh
$git clone https://github.com/tooBusyNow/custom-net-tools
``` 
After that you may navigate to `cmd` directory and start godns server using:
```sh
$./godns.exe
```

So, now you have a local DNS-Resolver up and running and can easily test it with standart `nslookup` tool:
```sh
$nslookup -type=A google.com 127.0.0.1
```

Or:

```sh
$nslookup -type=NS habr.ru 127.0.0.1
```

## Notes and features
Current version of resolver supports only 3 main types of DNS resource records: `NS`, `A`, `AAAA`

There is also a support for hot config reload. Typical use case is changing one root server to another. For instance, you can change `f-root server (192.5.5.241)` to `k-root (193.0.14.129)` without necessity of restart.

## Demostration:
