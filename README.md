# mon proxy
### exposes multiple read only sockets
I came up with because we neede to monitor multi tenant docker hosts with zabbix. 
We used zabbix agent 2 running in a container itself mounting the filtered sockets. 
Zabbix by itself is unable to do this internal
