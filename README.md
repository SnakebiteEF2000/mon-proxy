# Mon-Proxy
Came up with this because we needed to monitor containers using zabbix Agent 2. Our Docker hosts are multi-tenant, which can't be done nicely in zabbix directly. 
This is why this proxy filters based on labels and output multiple sockets. 
We run multiple zabbix agents, which correlate to multiple hosts in zabbix so it's easy to do permission management.
