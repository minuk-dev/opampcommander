---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
type: docs
description: >
  Learn how to get started with OpAMP Commander.
---

## What is OpAMP Commander?

OpAMP Commander is an agent management system that implements the OpenTelemetry Agent Management Protocol (OpAMP). It provides the following features:

- Remote agent monitoring and management
- Centralized configuration management
- Real-time agent status monitoring
- Agent updates and deployment

## Prerequisites

Before using OpAMP Commander, you'll need:

- Go 1.21 or later
- Docker (optional, for container deployment)
- MongoDB (for data storage)

## Installation

### Building from Source

```bash
git clone https://github.com/opampcommander/opampcommander.git
cd opampcommander
make build
```

### Running with Docker

```bash
docker-compose up -d
```

## First Run

To run OpAMP Commander:

```bash
./opampcommander server
```

By default, the server runs at `http://localhost:8080`.

## Next Steps

- Learn how to configure OpAMP Commander in the [Configuration Guide](/en/docs/configuration/)
- Check out API usage in the [API Reference](/en/docs/api/)
- Explore real-world use cases through [Examples](/en/docs/examples/)
