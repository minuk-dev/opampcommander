---
title: "OpAMP Protocol Sequence"
description: "OpAMP protocol execution flow and callback sequence"
weight: 3
---

# OpAMP Protocol Execution Sequence

## Agent Connection and Message Exchange Flow

```mermaid
sequenceDiagram
    participant Agent as OpAMP Agent
    participant Controller as opamp.Controller
    participant OpAMPLib as OpAMP Server Lib
    participant Service as opamp.Service
    participant AgentUC as AgentUsecase
    participant ConnUC as ConnectionUsecase
    participant WS as WebSocketRegistry
    participant DB as MongoDB Repository

    Note over Agent,DB: Connection Phase
    
    Agent->>Controller: GET/POST /api/v1/opamp
    Controller->>OpAMPLib: handler(writer, request)
    OpAMPLib->>Controller: OnConnecting(request)
    Controller-->>OpAMPLib: ConnectionResponse {Accept: true}
    OpAMPLib-->>Agent: Connection Accepted
    
    OpAMPLib->>Service: OnConnected(ctx, connection)
    Service->>Service: detectConnectionType(conn)
    Service->>Service: NewConnection(conn, type)
    alt WebSocket
        Service->>WS: Register(connID, wsConn)
    end
    Service->>ConnUC: SaveConnection(ctx, connection)
    ConnUC->>DB: Save
    
    Note over Agent,DB: Message Exchange Phase
    
    loop Active Connection
        Agent->>OpAMPLib: AgentToServer Message
        Note right of Agent: - InstanceUID<br/>- AgentDescription<br/>- Health<br/>- Capabilities<br/>- EffectiveConfig<br/>- RemoteConfigStatus<br/>- PackageStatuses
        
        OpAMPLib->>Service: OnMessage(ctx, conn, message)
        Service->>ConnUC: GetConnectionByID(ctx, conn)
        ConnUC->>DB: Find
        DB-->>ConnUC: connection
        ConnUC-->>Service: connection
        
        Service->>Service: Update connection.InstanceUID
        alt WebSocket
            Service->>WS: UpdateInstanceUID(connID, instanceUID)
        end
        Service->>ConnUC: SaveConnection(ctx, connection)
        ConnUC->>DB: Save
        
        Service->>AgentUC: GetOrCreateAgent(ctx, instanceUID)
        AgentUC->>DB: Find/Create
        DB-->>AgentUC: agent
        AgentUC-->>Service: agent
        Service->>Service: Update agent.Status (Connected=true)
        
        Service->>Service: report(agent, message, server)
        Note right of Service: Update agent from message:<br/>- AgentDescription<br/>- Health<br/>- RemoteConfigStatus<br/>- PackageStatuses<br/>- EffectiveConfig
        
        Service->>AgentUC: SaveAgent(ctx, agent)
        AgentUC->>DB: Save
        
        Service->>Service: fetchServerToAgent(ctx, agent)
        Note right of Service: Build response:<br/>- RemoteConfig<br/>- ConnectionSettings<br/>- PackagesAvailable<br/>- Flags<br/>- Command
        
        Service-->>OpAMPLib: ServerToAgent Response
        OpAMPLib-->>Agent: ServerToAgent Message
        Note left of Agent: - InstanceUID<br/>- RemoteConfig<br/>- ConnectionSettings<br/>- PackagesAvailable<br/>- Flags<br/>- Command
    end
    
    Note over Agent,DB: Error Handling (if occurs)
    
    opt Read Error
        OpAMPLib->>Service: OnReadMessageError(conn, type, msg, err)
        Note right of Service: Log error details
    end
    
    opt Response Error
        OpAMPLib->>Service: OnMessageResponseError(conn, msg, err)
        Note right of Service: Log error details
    end
    
    Note over Agent,DB: Disconnection Phase
    
    Agent->>OpAMPLib: Disconnect
    OpAMPLib->>Service: OnConnectionClose(connection)
    Service->>Service: Queue background cleanup
    Note right of Service: Return immediately
    
    Service->>Service: Background Loop executes
    Service->>ConnUC: GetConnectionByID(ctx, conn)
    ConnUC->>DB: Find
    DB-->>ConnUC: connection
    ConnUC-->>Service: connection
    
    alt Not Anonymous
        Service->>AgentUC: GetAgent(ctx, instanceUID)
        AgentUC->>DB: Find
        DB-->>AgentUC: agent
        AgentUC-->>Service: agent
        Service->>Service: Update agent.Status (Connected=false)
        Service->>AgentUC: SaveAgent(ctx, agent)
        AgentUC->>DB: Save
    end
    Service->>ConnUC: DeleteConnection(ctx, connection)
    ConnUC->>DB: Delete
    Note right of WS: WebSocket registry<br/>auto-cleanup
```

## OpAMP Callbacks Summary

| Callback | When Called | Purpose |
|----------|-------------|---------|
| **OnConnecting** | Before accepting connection | Validate and decide whether to accept |
| **OnConnected** | After connection established | Initialize connection tracking |
| **OnMessage** | When message received from agent | Process agent data and send response |
| **OnReadMessageError** | When failed to read message | Log and handle read errors |
| **OnMessageResponseError** | When failed to send response | Log and handle send errors |
| **OnConnectionClose** | When connection closed | Cleanup resources asynchronously |

## Architecture Layers

The diagram follows Hexagonal Architecture (Ports and Adapters):

### Primary Adapter (Infrastructure - In)
- **opamp.Controller**: HTTP/WebSocket endpoint handler
- **OpAMP Server Lib**: OpAMP protocol implementation library

### Application Layer
- **opamp.Service**: OpAMP protocol callback implementation
- **AgentUsecase**: Agent business logic
- **ConnectionUsecase**: Connection management logic

### Secondary Adapter (Infrastructure - Out)
- **WebSocketRegistry**: WebSocket connection registry
- **MongoDB Repository**: Data persistence

## Connection Types

- **WebSocket**: Bidirectional, persistent, registered in WebSocketRegistry
- **HTTP**: Unidirectional, request-response, not registered

---

## Server-Initiated Command Flow (SendCommand)

This flow shows how the server sends commands to agents, such as updating agent configuration.

```mermaid
sequenceDiagram
    participant Admin as Administrator/API
    participant Controller as agent.Controller
    participant AppService as agent.Service
    participant DomainService as domain.AgentService
    participant Agent as model.Agent
    participant CommandUC as CommandUsecase
    participant WS as WebSocketRegistry
    participant WSConn as WebSocketConnection
    participant OpAMPAgent as OpAMP Agent

    Note over Admin,OpAMPAgent: Command Initiation Phase
    
    Admin->>Controller: POST /api/v1/agents/{id}/send-command
    Note right of Admin: Request body contains<br/>command data
    Controller->>AppService: SendCommand(ctx, instanceUID, command)
    
    Note over AppService,Agent: Application Layer Processing
    
    AppService->>DomainService: UpdateAgentConfig(ctx, instanceUID, config)
    DomainService->>DomainService: GetOrCreateAgent(ctx, instanceUID)
    DomainService->>Agent: ApplyRemoteConfig(config)
    Note right of Agent: Agent model updates:<br/>1. NewRemoteConfigData(config)<br/>2. Apply to agent.Spec.RemoteConfig<br/>3. Add RemoteConfigUpdated command<br/>4. Command includes config hash
    
    DomainService->>DomainService: SaveAgent(ctx, agent)
    
    Note over DomainService,WSConn: Check if agent is connected
    
    alt Agent is connected via WebSocket
        DomainService->>DomainService: agent.IsConnected(ctx)
        DomainService->>DomainService: agent.HasPendingServerMessages()
        
        Note over DomainService,OpAMPAgent: Send Server Message
        
        DomainService->>DomainService: serverMessageUsecase.SendMessageToServer()
        Note right of DomainService: In standalone mode,<br/>message goes to in-memory channel
        
        Note over WS,OpAMPAgent: Message Processing (async)
        
        DomainService->>WS: GetByInstanceUID(instanceUID)
        WS-->>DomainService: wsConn
        
        DomainService->>WSConn: Send(ctx, ServerToAgent message)
        Note right of WSConn: ServerToAgent contains:<br/>- RemoteConfig<br/>- Flags<br/>- Command
        
        WSConn->>OpAMPAgent: Push ServerToAgent via WebSocket
        OpAMPAgent-->>WSConn: ACK
    else Agent is not connected (HTTP/Polling)
        Note over DomainService: Message will be delivered<br/>on next agent poll
    end
    
    Note over AppService,CommandUC: Audit Logging
    
    AppService->>CommandUC: SaveCommand(ctx, command)
    Note right of CommandUC: Save command for audit trail
    
    DomainService-->>AppService: Success
    AppService-->>Controller: Success
    Controller-->>Admin: 200 OK
    
    Note over OpAMPAgent: Agent applies new configuration
```

### SendCommand Flow Details

#### 1. API Request
- Administrator calls REST API endpoint to send command to agent
- Request includes target agent instance UID and command data

#### 2. Application Layer
- **agent.Service.SendCommand**: Orchestrates the command sending process
  - Calls domain service to update agent config
  - Saves command for audit log

#### 3. Domain Layer
- **AgentService.UpdateAgentConfig**: Business logic for config updates
  - Gets or creates agent model
  - Applies remote config to agent model
  - Saves agent (triggers server message if connected)

#### 4. Agent Model
- **Agent.ApplyRemoteConfig**: Updates agent's remote configuration
  - Creates RemoteConfigData from the provided config using `NewRemoteConfigData()`
  - Marshals config to YAML and computes hash
  - Applies RemoteConfigData to agent's RemoteConfig
  - Adds RemoteConfigUpdated command to agent's Commands list
  - Command includes config hash for tracking

**Key Types** (all in `model` package):
- `RemoteConfig`: Manages remote configuration state
- `RemoteConfigData`: Contains config bytes, hash, and status
- `RemoteConfigStatus`: Config application status (Unset, Applied, Applying, Failed)
- `AgentCommand`: Command to be sent to agent (includes RemoteConfigUpdated flag)

#### 5. Message Delivery (if connected)
- Checks if agent has active WebSocket connection
- Retrieves WebSocket connection from registry
- Sends ServerToAgent message through WebSocket
- Message includes:
  - Remote configuration
  - Flags (e.g., RemoteConfigChanged)
  - Commands

#### 6. Agent Processing
- Agent receives ServerToAgent message
- Applies new configuration
- Sends back status in next AgentToServer message

### Key Components

| Component | Layer | Responsibility |
|-----------|-------|----------------|
| **agent.Controller** | Primary Adapter | HTTP endpoint handler |
| **agent.Service** | Application Layer | Command orchestration |
| **AgentService** | Domain Layer | Agent business logic |
| **CommandUsecase** | Domain Layer | Command audit logging |
| **WebSocketRegistry** | Secondary Adapter | WebSocket connection management |
| **WebSocketConnection** | Secondary Adapter | WebSocket communication |
| **model.Agent** | Domain Model | Agent state including RemoteConfig and Commands |
| **model.RemoteConfig** | Domain Model | Remote configuration management |
| **model.RemoteConfigData** | Domain Model | Config data with hash and status |

### Domain Model Structure

```
Agent
├── Metadata (InstanceUID, Description, Capabilities)
├── Spec
│   └── RemoteConfig (RemoteConfigData, LastErrorMessage)
├── Commands ([]AgentCommand)
│   └── RemoteConfigUpdated, ReportFullState, etc.
└── Status (EffectiveConfig, Health, PackageStatuses)
```

### Message Delivery Modes

- **WebSocket (Push)**: Server immediately pushes configuration to connected agents
- **HTTP (Poll)**: Configuration is delivered when agent next polls the server
- **Hybrid**: Agents can switch between WebSocket and HTTP as needed

### Configuration Update Flow

1. **API Request** → Creates command with config data
2. **Application Layer** → Validates and orchestrates
3. **Domain Layer** → 
   - Calls `agent.ApplyRemoteConfig(config)`
   - Creates `RemoteConfigData` with hash
   - Updates `agent.Spec.RemoteConfig`
   - Adds `RemoteConfigUpdated` to `agent.Commands`
4. **Persistence** → Saves agent with new config and pending command
5. **Message Delivery** → 
   - Retrieves agent from WebSocketRegistry
   - Sends ServerToAgent message with RemoteConfig
   - Agent receives and applies configuration
6. **Status Report** → Agent sends back status in next AgentToServer message
