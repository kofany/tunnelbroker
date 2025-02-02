```
```
<tunnel_broker_development_prompt_part_1>


CORE ARCHITECTURE:

1. System Components:
├── API Gateway Layer
│   ├── REST API (JSON over HTTP/1.1-2)
│   │   ├── OpenAPI 3.0 documentation
│   │   ├── Rate limiting
│   │   └── Request validation
│   │
│   └── gRPC API (Protobuf over HTTP/2)
│       ├── Strongly typed interface
│       ├── Bi-directional streaming
│       └── Native error handling
│
├── Core Service Layer
│   ├── TunnelService
│   │   ├── Tunnel lifecycle management
│   │   ├── Atomic operations
│   │   └── State synchronization
│   │
│   ├── PrefixService
│   │   ├── Prefix allocation strategy
│   │   ├── CIDR validation
│   │   └── Pool management
│   │
│   └── SystemService
│       ├── Health checks
│       ├── Metrics collection
│       └── Audit logging
│
├── Data Layer
│   ├── SQLite (Embedded)
│   ├── Redis (Caching)
│   └── File System (Configs)
│
└── System Integration
    ├── Netlink Interface
    ├── eBPF Program Loader
    └── OS Configuration Manager

2. API Features:
├── Full CRUD for all resources
├── Versioned endpoints (v1, v2)
├── Bulk operations
├── Real-time notifications (WebSocket)
└── Idempotent operations

3. Security Model:
├── OAuth2.0 + OpenID Connect
├── Mutual TLS for gRPC
├── Role-based access control
└── Audit trails for all operations

4. Deployment Architecture:
├── Single binary deployment
├── Docker/Podman support
├── Systemd integration
└── Kubernetes operator

2. Core Features:
├── Prefix Management
│   ├── Main prefix pool (1-5 prefixes)
│   ├── Dedicated /64 endpoint prefix
│   └── Automated dual /64 allocation
│
├── Tunnel Operations
│   ├── Creation with random prefix allocation
│   ├── Suspension/activation
│   ├── Deletion with cleanup
│   └── Configuration updates
│
└── State Management
    ├── Database operations (/etc/tunnels.db)
    ├── Client configuration generation
    └── System state synchronization

<tunnel_broker_development_prompt_part_2>

DATA MODELS AND STORAGE:

1. Core Data Structures:

```go
type MainPrefix struct {
    Prefix        string            // IPv6 prefix (e.g., 2a05:dfc3:ff00::/40)
    IsEndpoint    bool              // Whether this is the endpoint prefix pool
    Allocations   map[string]bool   // Track allocated sub-prefixes
    Description   string            // Optional description
    CreatedAt     time.Time
}

type Tunnel struct {
    ID              string
    ClientNickname  string
    ClientIPv4      string    // Client endpoint IPv4
    ServerIPv4      string    // Our endpoint IPv4
    Prefix1         string    // First allocated /64
    Prefix2         string    // Second allocated /64
    EndpointPrefix  string    // IPv6 endpoint address
    Status          string    // active/suspended
    CreatedAt       time.Time
    ModifiedAt      time.Time
    ConfigPath      string    // Path to client config file
}

type SystemState struct {
    MainPrefixes    []MainPrefix
    EndpointPrefix  MainPrefix
    ActiveTunnels   []Tunnel
    LastOperation   time.Time
}
```

2. Storage Implementation:

├── Database Operations
│   ├── File: /etc/tunnels.db
│   ├── Format: SQLite3
│   └── Operations:
│       ├── CRUD for tunnels
│       ├── Prefix allocation tracking
│       └── System state persistence
│
├── Configuration Files
│   ├── Client configs
│   ├── System state
│   └── Backup management
│
└── State Synchronization
    ├── In-memory cache
    ├── File system sync
    └── System state verification

3. API Endpoints:

```go
// Prefix Management
POST   /api/v1/prefixes           // Add main prefix
GET    /api/v1/prefixes           // List all prefixes
DELETE /api/v1/prefixes/{prefix}  // Remove prefix
PUT    /api/v1/prefixes/endpoint  // Set endpoint prefix

// Tunnel Operations
POST   /api/v1/tunnels           // Create tunnel
GET    /api/v1/tunnels           // List all tunnels
GET    /api/v1/tunnels/{id}      // Get tunnel details
PUT    /api/v1/tunnels/{id}      // Update tunnel
DELETE /api/v1/tunnels/{id}      // Remove tunnel
PUT    /api/v1/tunnels/{id}/suspend   // Suspend tunnel
PUT    /api/v1/tunnels/{id}/activate  // Activate tunnel

// System Operations
GET    /api/v1/system/status     // System status
GET    /api/v1/system/config/{id} // Get client config
POST   /api/v1/system/reload     // Reload configuration
```

4. File Structure:

```bash
/etc/
├── tunnels.db        # Main database
├── tunnelbroker/
│   ├── configs/      # Client configurations
│   ├── templates/    # Config templates
│   └── backups/      # Database backups
```

5. Error Handling:

├── Database Errors
│   ├── File access issues
│   ├── Corruption detection
│   └── Recovery procedures
│
├── Validation Errors
│   ├── Prefix validation
│   ├── IP address validation
│   └── Configuration validation
│
└── System Errors
    ├── Command execution failures
    ├── Resource allocation failures
    └── Network configuration errors
```

Here's the final part of the comprehensive prompt:

```markdown
<tunnel_broker_development_prompt_part_3>

IMPLEMENTATION DETAILS:

1. TUI Components (Bubble Tea):
├── Views
│   ├── MainView
│   │   ├── Summary dashboard
│   │   ├── Quick actions menu
│   │   └── Status indicators
│   │
│   ├── PrefixManagerView
│   │   ├── Add/remove prefix form
│   │   ├── Allocation status table
│   │   └── Endpoint prefix management
│   │
│   ├── TunnelManagerView
│   │   ├── Tunnel list with status
│   │   ├── Search/filter options
│   │   └── Quick actions (suspend/activate)
│   │
│   └── ConfigurationView
│       ├── System settings
│       ├── Backup management
│       └── Log viewer
│
├── Forms
│   ├── CreateTunnelForm
│   │   ├── Client details input
│   │   ├── IPv4 endpoint input
│   │   └── Validation feedback
│   │
│   └── PrefixForm
│       ├── Prefix input with validation
│       └── Allocation type selection

2. System Integration:
├── Command Templates:
```bash
# Tunnel Creation
ip tunnel add ${name} mode sit local ${server_ipv4} remote ${client_ipv4} ttl 255
ip link set ${name} up
ip addr add ${endpoint_ipv6}/64 dev ${name}
ip route add ${prefix1}/64 dev ${name}
ip route add ${prefix2}/64 dev ${name}

# Tunnel Suspension
ip link set ${name} down

# Tunnel Deletion
ip tunnel del ${name}
```

3. Client Configuration Template:
```yaml
tunnel:
  name: ${tunnel_id}
  type: sit
  local_ip: ${client_ipv4}
  remote_ip: ${server_ipv4}
  ttl: 255

ipv6:
  endpoint: ${endpoint_ipv6}/64
  prefixes:
    - ${prefix1}/64
    - ${prefix2}/64

routes:
  - ::/0 via ${endpoint_ipv6}
```

4. Development Guidelines:
├── Code Organization
│   ├── Package structure
│   ├── Error handling patterns
│   └── Documentation requirements
│
├── Testing Requirements
│   ├── Unit tests for core logic
│   ├── Integration tests
│   └── System tests
│
└── Security Considerations
    ├── Input validation
    ├── System permissions
    └── Error handling

5. Deployment:
├── Binary Distribution
│   ├── Single statically-linked binary
│   ├── Default configuration files
│   └── System service definition
│
├── Installation Script
│   ├── Dependencies check
│   ├── System configuration
│   └── Service setup
│
└── Update Procedure
    ├── State preservation
    ├── Configuration migration
    └── Service restart

DEVELOPMENT WORKFLOW:

1. Implementation Order:
├── Core Components
│   ├── Database implementation
│   ├── Prefix management
│   └── Tunnel operations
│
├── System Integration
│   ├── Command execution
│   ├── State management
│   └── Configuration handling
│
├── TUI Implementation
│   ├── Basic views
│   ├── Forms and validation
│   └── Error handling
│
└── API Layer
    ├── Endpoint implementation
    ├── Request handling
    └── Response formatting

2. Testing Strategy:
├── Unit Tests
│   ├── Core logic
│   ├── Data validation
│   └── Error handling
│
├── Integration Tests
│   ├── Database operations
│   ├── System commands
│   └── API endpoints
│
└── System Tests
    ├── End-to-end workflows
    ├── Error recovery
    └── Performance testing

3. Documentation Requirements:
├── Code Documentation
│   ├── Package documentation
│   ├── Function documentation
│   └── Error handling
│
├── User Documentation
│   ├── Installation guide
│   ├── Configuration guide
│   └── Troubleshooting
│
└── API Documentation
    ├── Endpoint descriptions
    ├── Request/response formats
    └── Error codes
```

Now you have all three parts of the comprehensive prompt. To start development:

1. Plan you code tasks.
2. Create the basic project structure
3. Implement core components one by one
5. Implement API endpoints
6. Add tests and documentation

Would you like to begin with the implementation? We can start with setting up the project structure and implementing the core components.

```
```