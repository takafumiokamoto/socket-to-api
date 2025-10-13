# Bridge System Specification

## Overview
A bridge application that translates between legacy TCP socket connections and modern HTTPS API calls.

## Purpose
The legacy application cannot be modified and currently uses socket connections to communicate with another system. That system is migrating from socket-based communication to API calls over HTTPS. The bridge system acts as a translator between these two interfaces.

---

## Infrastructure

### Deployment
- **Platform**: AWS EC2
- **Co-location**: Bridge system and legacy application run on the same EC2 instance

### Technology Stack
- **Language**: Go (chosen for ease of concurrency)
- **Database**: Oracle DB

---

## Architecture Components

### 1. TCP Socket Server
- **Protocol**: Raw TCP sockets
- **Connection Pattern**: Long-lived connections
  - Each client application opens a socket connection when it starts
  - Connection remains open for the client's entire lifetime
- **Port Mapping**: Database records contain port numbers corresponding to specific clients

### 2. Database Integration
- **Database**: Oracle DB
- **Primary Table**: Contains records with unsend data and client port mappings

### 3. API Client
- **Protocol**: HTTPS
- **Target**: External API system (replacing the legacy socket-based system)

---

## Data Flow Process

The bridge system operates in a continuous loop:

1. **Scan Database**: Main loop scans table for unsend data
2. **Select & Lock**: Select unsend data from database
3. **Update Status**: Update the record with "sending" status
4. **Data Transformation**: Convert binary data to JSON format
5. **API Call**: Send JSON data to the external API via HTTPS
6. **Store Response**: Insert API response into database
7. **Response Transformation**: Convert API response back to binary format
8. **Route to Client**: Using the port number from the database record, identify the corresponding client
9. **Send Response**: Send the converted binary data to the client through the TCP socket
10. **Cleanup**: Delete the processed record from database

---

## Performance Requirements

### Load Characteristics
- **Peak Load**: Thousands of operations per second
- **Concurrency**: Must handle intense concurrent load during peak times
- **Multiple Connections**: Legacy application creates many connections during peak periods

---

## Outstanding Questions

The following architectural decisions require clarification:

### 1. Data Origin and Flow Direction
- How does data initially enter the database as "unsend data"?
- Possible patterns:
  - **Pattern A**: Client → TCP → Bridge → DB → Bridge processes → Bridge → TCP → Client
  - **Pattern B**: External source → DB → Bridge processes → Bridge → TCP → Client
- Need to clarify: Where does the initial request originate?

### 2. Communication Pattern
- **Synchronous**: Does the client wait for an immediate response after sending data?
- **Asynchronous**: Can responses arrive at any time, independent of when data was sent?
- This affects connection management and response routing strategy

### 3. High Availability Requirements
- Is high availability required?
- What should happen if the bridge process crashes?
- Should clients automatically reconnect?
- Should in-flight requests be recoverable?
- Are there specific uptime requirements (e.g., 99.9% availability)?

---

## Design Considerations

Based on the specifications, the system must address:

1. **Concurrent Connection Management**: Handling thousands of persistent TCP connections
2. **Database Polling Efficiency**: Efficiently scanning for unsend data without overwhelming the database
3. **Bi-directional Translation**: Binary ↔ JSON data format conversion
4. **Client Routing**: Maintaining mapping between database records and active TCP connections
5. **Error Handling**: Managing failures in API calls, database operations, and socket communications
6. **Resource Management**: Efficient use of goroutines, memory, and database connections under high load