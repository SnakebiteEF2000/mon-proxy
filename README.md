# Mon-Proxy

[![Go Report Card](https://goreportcard.com/badge/github.com/SnakebiteEF2000/mon-proxy)](https://goreportcard.com/badge/github.com/SnakebiteEF2000/mon-proxy)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

Mon-Proxy is a powerful tool designed to monitor Docker containers using Zabbix Agent 2 in multi-tenant environments. It provides a solution for filtering Docker API requests based on container labels, enabling fine-grained access control and monitoring capabilities.

## 🌟 Features

- Filter Docker API requests based on container labels
- Support for multiple output sockets
- Seamless integration with Zabbix Agent 2
- Easy permission management for multi-tenant environments
- Configurable through environment variables

## 🚀 Getting Started

### Prerequisites

- Docker
- Go 1.16 or higher

### Installation

Clone the repository:

```bash
git clone https://github.com/yourusername/mon-proxy.git
cd mon-proxy
```

Build the Docker image:

```bash
docker build -t mon-proxy:latest .
```

### Configuration

Mon-Proxy is configured using environment variables. The main configuration file can be found in:

```go
func GetSocketConfigs() []proxy.ProxyConfig {
    var configs []proxy.ProxyConfig

    loop:
    for i := 1; ; i++ {
        destSocket := cfg.GetEnv(fmt.Sprintf("DESTINATION_SOCKET_%d", i), "")
        label := cfg.GetEnv(fmt.Sprintf("REQUIRED_LABEL_%d", i), "")

        if destSocket == "" || label == "" {
            break loop
        }

        configs = append(configs, proxy.ProxyConfig{
            DestinationSocket: destSocket,
            RequiredLabel:     label,
        })
        log.Debug("destination socket config: ", destSocket, "label: ", label)
    }
    return configs
}
```

### Key environment variables:

- `SOURCE_SOCKET`: The source Docker socket (default: `/var/run/docker.sock`)
- `DESTINATION_SOCKET_n`: The destination socket for the nth configuration
- `REQUIRED_LABEL_n`: The required label for the nth configuration

## 🏗️ Usage

You can run Mon-Proxy using Docker Compose. An example configuration is provided below:

```yaml
services:
  mon-proxy:
    image: mon-proxy:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - sock:/run/mon-proxy
    environment:
      - SOURCE_SOCKET=/var/run/docker.sock
      - DESTINATION_SOCKET_1=/run/mon-proxy/filtered-docker-1.sock
      - REQUIRED_LABEL_1=it.monitoring.enabled=true
      - DESTINATION_SOCKET_2=/run/mon-proxy/filtered-docker-2.sock
      - REQUIRED_LABEL_2=tenant.monitoring.enabled=false
    restart: unless-stopped

  test-client:
    image: docker:dind
    depends_on:
      - mon-proxy
    volumes:
      - sock:/run/mon-proxy:ro
    environment:
      - DOCKER_HOST=unix:///run/mon-proxy/filtered-docker-1.sock
    command: ["/bin/sleep", "Infinity"]

  allowed:
    image: alpine:latest
    depends_on:
      - test-client
    labels:
      - monitoring.enabled=true
      - it.monitoring.enabled=true
    command: ["/bin/sleep", "Infinity"]

  denied:
    image: alpine:latest
    depends_on:
      - test-client
    labels:
      - monitoring.enabled=false
    command: ["/bin/sleep", "Infinity"]

volumes:
  sock:
```

To start the services:

```bash
docker-compose up -d
```

This will start Mon-Proxy along with test containers to demonstrate its functionality.

## 🧪 Testing

You can test the proxy by using the `test-client` service in the Docker Compose file. It's configured to use the filtered Docker socket.

## 🛠️ Development

To set up a development environment:

1. Install Go 1.16 or higher
2. Clone the repository
3. Run `go mod download` to install dependencies

You can use the provided VS Code launch configuration for debugging:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Package",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/mon-proxy"
        }
    ]
}
```

## 📜 License

This project is licensed under the GNU General Public License v3.0. See the LICENSE file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📞 Support

If you have any questions or need support, please open an issue in the GitHub repository.

---

We hope you find Mon-Proxy useful for your Docker monitoring needs in multi-tenant environments!
