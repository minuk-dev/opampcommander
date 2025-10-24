---
title: "Configuration"
linkTitle: "Configuration"
weight: 2
type: docs
description: >
  Learn how to configure OpAMP Commander.
---

## Basic Configuration

OpAMP Commander can be configured through a YAML file. The default configuration file is located at `configs/config.yaml`.

### Server Configuration

```yaml
server:
  http:
    address: ":8080"
  grpc:
    address: ":9090"
```

### OpAMP Configuration

```yaml
opamp:
  server:
    endpoint: "ws://localhost:4320/v1/opamp"
```

### Database Configuration

```yaml
database:
  mongodb:
    uri: "mongodb://localhost:27017"
    database: "opampcommander"
```

## Environment Variables

Configuration values can also be provided via environment variables:

- `SERVER_HTTP_ADDRESS`: HTTP server address
- `SERVER_GRPC_ADDRESS`: gRPC server address
- `MONGODB_URI`: MongoDB connection URI
- `MONGODB_DATABASE`: MongoDB database name

## Advanced Configuration

### Logging

```yaml
logging:
  level: "info"
  format: "json"
```

### Authentication

```yaml
auth:
  enabled: true
  jwt:
    secret: "your-secret-key"
```
