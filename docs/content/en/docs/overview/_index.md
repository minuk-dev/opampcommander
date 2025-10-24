---
title: "Overview"
linkTitle: "Overview"
weight: -1
type: docs
description: >
  An overview of OpAMP Commander and its capabilities.
---

## What is OpAMP Commander?

OpAMP Commander is a comprehensive management platform for OpenTelemetry agents that implements the Open Agent Management Protocol (OpAMP). It provides a centralized solution for managing, monitoring, and configuring distributed telemetry collection agents.

## Key Features

### Centralized Management
Manage all your OpenTelemetry agents from a single web-based interface. Monitor agent health, status, and performance metrics in real-time.

### Dynamic Configuration
Update agent configurations dynamically without restarting agents. Push configuration changes to individual agents or groups of agents instantly.

### Agent Discovery
Automatically discover and register new agents as they come online. Track agent inventory and deployment status across your infrastructure.

### Version Control
Manage agent versions and coordinate upgrades across your fleet. Roll out updates gradually with canary deployments.

### Security
Built-in authentication and authorization ensure secure communication between the server and agents. Support for TLS/SSL encryption and JWT-based authentication.

## Architecture

OpAMP Commander follows a modern, layered architecture designed for scalability and maintainability.

### System Overview

```mermaid
graph TB
    subgraph Clients
        CLI[opampctl CLI]
        Agent[OpAMP Agents<br/>OpenTelemetry Collectors]
        WebUI[Web UI<br/>Planned]
    end

    subgraph "OpAMP Commander Server"
        Server[OpAMP Commander<br/>API Server]
    end

    subgraph Storage
        DB[(MongoDB<br/>Database)]
    end

    CLI -->|HTTP/REST<br/>Management API| Server
    Agent -->|WebSocket/gRPC<br/>OpAMP Protocol| Server
    WebUI -.->|HTTP/REST| Server
    Server -->|Persist Data| DB

    style Server fill:#4a90e2,stroke:#333,stroke-width:2px,color:#fff
    style DB fill:#6c757d,stroke:#333,stroke-width:2px,color:#fff
    style CLI fill:#28a745,stroke:#333,stroke-width:2px,color:#fff
    style Agent fill:#ffc107,stroke:#333,stroke-width:2px,color:#000
    style WebUI fill:#17a2b8,stroke:#333,stroke-width:2px,color:#fff
```

### Component Interactions

**Agent Registration & Management:**
```mermaid
sequenceDiagram
    participant Agent as OpAMP Agent
    participant Server as OpAMP Commander
    participant DB as MongoDB

    Agent->>Server: Connect via WebSocket
    Server->>DB: Store Agent Info
    Server->>Agent: Send Configuration
    Agent->>Server: Report Status
    Server->>DB: Update Agent State
```

**Configuration Management via CLI:**
```mermaid
sequenceDiagram
    participant Admin as Administrator
    participant CLI as opampctl
    participant Server as OpAMP Commander
    participant Agent as OpAMP Agent

    Admin->>CLI: opampctl create agentgroup
    CLI->>Server: POST /api/v1/agentgroups
    Server->>CLI: Group Created
    
    Admin->>CLI: opampctl update config
    CLI->>Server: PUT /api/v1/agents/{id}/update-agent-config
    Server->>Agent: Push New Configuration
    Agent->>Server: Acknowledge
    Server->>CLI: Update Successful
```

### Key Features

- **REST API**: Full-featured HTTP API for programmatic management
- **OpAMP Protocol**: Native support for OpenTelemetry Agent Management Protocol
- **Agent Groups**: Manage multiple agents with shared configurations
- **Authentication**: JWT tokens, OAuth2 (GitHub), Basic Auth
- **Scalability**: Stateless design enabling horizontal scaling

### Technology Stack

- **Backend**: Go 1.21+, Gin web framework
- **Database**: MongoDB 4.4+ for persistence
- **Protocol**: OpAMP over WebSocket/gRPC
- **Observability**: Prometheus metrics, structured logging

## Use Cases

### Infrastructure Monitoring
Deploy and manage OpenTelemetry collectors across your infrastructure for comprehensive observability.

### Microservices Observability
Configure distributed tracing and metrics collection for containerized applications.

### Compliance and Governance
Enforce standardized telemetry collection policies across your organization.

### Cost Optimization
Control sampling rates and filter telemetry data to optimize storage and processing costs.

## Getting Started

Ready to start using OpAMP Commander? Check out our [Getting Started Guide](/en/docs/getting-started/) for installation and setup instructions.
