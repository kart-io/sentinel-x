# sentinel-x

Sentinel-X is a distributed intelligent operations system.

## Getting Started

### Prerequisites

*   Go 1.25.3 or higher

### Build

To build the project components:

```bash
go build ./cmd/...
```

### Development

This project uses a **Monorepo** structure similar to Kubernetes. The core agent library (`goagent`) is included directly in `staging/src/github.com/kart-io/goagent`.

For detailed instructions on how to contribute, modify code, and sync dependencies, please refer to the **[Development Guide](DEVELOPMENT.md)**.

## License

See [LICENSE](LICENSE) file.