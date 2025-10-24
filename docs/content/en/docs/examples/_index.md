---
title: "Examples"
linkTitle: "Examples"
weight: 4
type: docs
description: >
  Explore OpAMP Commander usage examples.
---

## Basic Agent Connection

Example of connecting a simple OpAMP agent to OpAMP Commander.

### Go Agent Example

```go
package main

import (
    "context"
    "github.com/open-telemetry/opamp-go/client"
    "log"
)

func main() {
    // Create OpAMP client
    opampClient := client.NewWebSocket(nil)
    
    // Connect to server
    err := opampClient.Start(context.Background(), client.StartSettings{
        OpAMPServerURL: "ws://localhost:4320/v1/opamp",
        InstanceUid:    "agent-001",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    defer opampClient.Stop(context.Background())
    
    // Keep program running
    select {}
}
```

## Configuration Update Example

Example of updating agent configuration using the REST API.

### cURL Example

```bash
curl -X PUT \
  http://localhost:8080/api/v1/agents/agent-001/config \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -d '{
    "config": {
      "receivers": {
        "otlp": {
          "protocols": {
            "grpc": {
              "endpoint": "0.0.0.0:4317"
            }
          }
        }
      }
    }
  }'
```

## Docker Compose Example

Example of running OpAMP Commander with MongoDB.

```yaml
version: '3.8'

services:
  opampcommander:
    image: opampcommander:latest
    ports:
      - "8080:8080"
      - "4320:4320"
    environment:
      - MONGODB_URI=mongodb://mongodb:27017
      - MONGODB_DATABASE=opampcommander
    depends_on:
      - mongodb

  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db

volumes:
  mongodb_data:
```
