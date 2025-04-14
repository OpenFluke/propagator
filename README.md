# Checking whats running on server

```
podman ps --format "table {{.Names}}\t{{.Ports}}"
```

NAMES PORTS
bio1 0.0.0.0:10001-10002->10001-10002/tcp, 0.0.0.0:10000->10000/udp
bio2 0.0.0.0:10004-10005->10004-10005/tcp, 0.0.0.0:10003->10003/udp
bio3 0.0.0.0:10007-10008->10007-10008/tcp, 0.0.0.0:10006->10006/udp
bio4 0.0.0.0:10010-10011->10010-10011/tcp, 0.0.0.0:10009->10009/udp
bio5 0.0.0.0:10013-10014->10013-10014/tcp, 0.0.0.0:10012->10012/udp
bio6 0.0.0.0:10016-10017->10016-10017/tcp, 0.0.0.0:10015->10015/udp
bio7 0.0.0.0:10019-10020->10019-10020/tcp, 0.0.0.0:10018->10018/udp
bio8 0.0.0.0:10022-10023->10022-10023/tcp, 0.0.0.0:10021->10021/udp
bio9 0.0.0.0:10025-10026->10025-10026/tcp, 0.0.0.0:10024->10024/udp
bio10 0.0.0.0:10028-10029->10028-10029/tcp, 0.0.0.0:10027->10027/udp
bio11 0.0.0.0:10030->10030/udp, 0.0.0.0:10031-10032->10031-10032/tcp
bio12 0.0.0.0:10033->10033/udp, 0.0.0.0:10034-10035->10034-10035/tcp
bio13 0.0.0.0:10037-10038->10037-10038/tcp, 0.0.0.0:10036->10036/udp
bio14 0.0.0.0:10040-10041->10040-10041/tcp, 0.0.0.0:10039->10039/udp
