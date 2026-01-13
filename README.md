# Optimal Truck Load Planner

> High-performance microservice for optimal truck load planning using Dynamic Programming

## Overview

SmartLoad is a stateless microservice that solves the multi-dimensional knapsack problem for logistics optimization. Given a truck's capacity constraints and a list of available orders, it finds the optimal combination that maximizes revenue while respecting weight, volume, route compatibility, and hazmat constraints.

### Key Features

- **Optimal Solutions** - Uses Dynamic Programming with bitmask for exact optimal results  
- **Sub-second Performance** - Handles 22 orders in under 800ms  
- **Production-Ready** - Comprehensive error handling, validation, and logging  
- **Clean Architecture** - SOLID principles, dependency injection, testable code  
- **Docker Native** - Multi-stage builds for small images  
- **Stateless Design** - No database required, fully horizontally scalable  

### Bonus Features (All 4 Implemented)

1. Dynamic Programming with Bitmask - O(2^n × n) optimal algorithm with precomputed incompatibility masks
2. Recursive Backtracking with Pruning - Alternative algorithm with branch-and-bound optimization
3. Pareto-Optimal Solutions - Returns multiple trade-off solutions (revenue vs utilization)
4. Configurable Weights - Customize optimization objectives (revenue/utilization/balanced)

## Quick Start

### Prerequisites

- Docker & Docker Compose (recommended)
- OR Go 1.21+ (for local development)

### Running with Docker

```bash
git clone https://github.com/aakashraj01/smart-load
cd smart-load

docker compose up --build
# Service will be available at http://localhost:8080
```

### Running Locally

```bash
# Install dependencies
go mod download

# Run the service
go run cmd/server/main.go
```

## API Documentation

### Base URL
```
http://localhost:8080
```

### Endpoints

#### Health Check
```bash
GET /healthz
GET /actuator/health
```

**Response:**
```json
{
  "status": "UP",
  "service": "SmartLoad Optimizer API",
  "version": "1.0.0"
}
```

#### Optimize Load
```bash
POST /api/v1/load-optimizer/optimize
Content-Type: application/json
```

**Request Body:**
```json
{
  "truck": {
    "id": "truck-123",
    "max_weight_lbs": 44000,
    "max_volume_cuft": 3000
  },
  "orders": [
    {
      "id": "ord-001",
      "payout_cents": 250000,
      "weight_lbs": 18000,
      "volume_cuft": 1200,
      "origin": "Los Angeles, CA",
      "destination": "Dallas, TX",
      "pickup_date": "2025-12-05",
      "delivery_date": "2025-12-09",
      "is_hazmat": false
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "truck_id": "truck-123",
  "selected_order_ids": ["ord-001", "ord-002"],
  "total_payout_cents": 430000,
  "total_weight_lbs": 30000,
  "total_volume_cuft": 2100,
  "utilization_weight_percent": 68.18,
  "utilization_volume_percent": 70.0
}
```

**Error Response (400):**
```json
{
  "error": {
    "code": 400,
    "message": "validation failed: truck max_weight_lbs must be positive"
  }
}
```

## Testing

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/load-optimizer/optimize \
  -H "Content-Type: application/json" \
  -d @sample-request.json
```

## Architecture & Design

### Tech Stack

- **Language:** Go 1.21
- **Web Framework:** Fiber v2
- **Algorithm:** Dynamic Programming with Bitmask
- **Architecture:** Clean Architecture (Layered)
- **Containerization:** Docker (Multi-stage build)

### Project Structure

```
smart-load/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   └── handlers.go          # HTTP handlers
│   ├── domain/
│   │   ├── models.go            # Domain models & types
│   │   └── constraints.go       # Business rules & validation
│   ├── service/
│   │   └── optimizer_service.go # Business logic orchestration
│   └── algorithm/
│       └── optimizer.go         # DP optimization algorithm
├── Dockerfile                   # Multi-stage Docker build
├── docker-compose.yml           # Service orchestration
├── sample-request.json          # Example API request
├── go.mod                       # Go dependencies
└── README.md                    # This file
```

### Design Principles

#### Clean Architecture (Layered)
```
┌─────────────────────────────────┐
│   API Layer (handlers.go)      │  ← HTTP, JSON, Fiber
├─────────────────────────────────┤
│   Service Layer (service.go)   │  ← Business orchestration
├─────────────────────────────────┤
│   Algorithm Layer (optimizer)  │  ← Pure DP algorithm
├─────────────────────────────────┤
│   Domain Layer (models.go)     │  ← Core entities & rules
└─────────────────────────────────┘
```

**Benefits:**
- Each layer has a single responsibility
- Easy to test in isolation
- Framework-agnostic domain logic
- Swappable algorithms

#### SOLID Principles

- **Single Responsibility:** Each struct/function does one thing
- **Open/Closed:** Easy to add new optimizers via interface
- **Liskov Substitution:** Optimizer interface allows swapping algorithms
- **Interface Segregation:** Small, focused interfaces
- **Dependency Inversion:** Depend on abstractions, not implementations

## Algorithm Details

### Dynamic Programming with Bitmask

**Problem:** Multi-dimensional 0/1 Knapsack with constraints  
**Time Complexity:** O(2^n * n)  
**Space Complexity:** O(2^n)  
**Optimal for:** n <= 22

#### How It Works

1. **State Representation:** Use a bitmask to represent which orders are selected
   - `0b0000` = no orders
   - `0b0101` = orders 0 and 2 selected
   - Total states: 2^n (e.g., 4,194,304 for n=22)

2. **DP Table:** `dp[mask]` stores the best state for that combination

3. **Transition:** For each state, try adding each remaining order
   ```
   for mask := 0 to 2^n - 1:
     for i := 0 to n - 1:
       if order i not in mask:
         newMask = mask | (1 << i)
         if canFit(order[i]) && compatible(order[i]):
           dp[newMask] = max(dp[newMask], dp[mask] + order[i])
   ```

4. **Result:** Find the state with maximum payout

#### Performance Optimizations

- **Preprocessing:** Filter orders that can't possibly fit
- **Early Pruning:** Skip invalid states immediately
- **Bit Operations:** Native CPU instructions (very fast)
- **O(1) Compatibility:** Precomputed bitmask for instant compatibility checks

## Performance Benchmarks

| # Orders | States (2^n) | Time (ms) | Memory (MB) |
|----------|-------------|-----------|-------------|
| 10       | 1,024       | <10       | ~5          |
| 15       | 32,768      | ~50       | ~10         |
| 20       | 1,048,576   | ~400      | ~50         |
| 22       | 4,194,304   | ~700      | ~100        |

**Target:** < 800ms for n=22

## Bonus Features (Detailed)

### 1. Dynamic Programming with Bitmask

**Status:** Fully implemented and optimized

**What It Is:**
- Classic solution for n ≤ 22 orders
- Uses bitmask to represent order combinations
- Guarantees optimal solution

**Optimizations:**
- **Precomputed Incompatibility Masks**: O(1) compatibility checks instead of O(N)
- **State Pruning**: Skips invalid states early
- **Bit Operations**: Native CPU instructions for speed

**Performance:**
- n=22: ~674ms (under 800ms target!)
- Space: O(2^n) = 4.2M states for n=22

**API Usage:**
```bash
# Default algorithm (no config needed)
curl -X POST http://localhost:8080/api/v1/load-optimizer/optimize \
  -H "Content-Type: application/json" \
  -d @sample-request.json
```

---

### 2. Recursive Backtracking with Pruning

**Status:** Fully implemented

**What It Is:**
- Alternative optimization algorithm using DFS
- Branch-and-bound pruning for efficiency
- Good for educational purposes and smaller problems

**Features:**
- Upper bound calculation for remaining orders
- Early termination when impossible to improve
- Explores solution space recursively

**API Usage:**
```bash
curl -X POST http://localhost:8080/api/v1/load-optimizer/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "truck": {...},
    "orders": [...],
    "optimization_config": {
      "algorithm": "backtracking"
    }
  }'
```

**Supported Algorithms:**
- `"dp"` - Dynamic Programming (default)
- `"backtracking"` - Recursive backtracking
- `"greedy"` - Fast approximation
- `"auto"` - Automatic selection

---

### 3. Pareto-Optimal Solutions

**Status:** Fully implemented

**What It Is:**
- Returns multiple non-dominated solutions
- Trade-off between revenue and utilization
- Helps carriers make informed decisions

**How It Works:**
1. Runs optimization with different objective weights (1.0/0.0, 0.8/0.2, 0.6/0.4, 0.4/0.6, 0.2/0.8)
2. Filters non-dominated solutions (Pareto frontier)
3. Returns solutions sorted by score

**API Usage:**
```bash
# Dedicated endpoint for Pareto solutions
curl -X POST http://localhost:8080/api/v1/load-optimizer/pareto-solutions \
  -H "Content-Type: application/json" \
  -d @sample-request.json
```

**Response:**
```json
{
  "truck_id": "truck-123",
  "count": 1,
  "solutions": [
    {
      "order_ids": ["ord-001", "ord-002"],
      "total_payout_cents": 630000,
      "utilization_weight_percent": 90.91,
      "utilization_volume_percent": 96.67,
      "score": 630000
    }
  ]
}
```

---

### 4. Configurable Weights & Objectives

**Status:** Fully implemented with validation

**What It Is:**
- Customize optimization priorities
- Balance revenue vs truck utilization
- Flexible objective functions

**Supported Objectives:**
- `"revenue"` - Maximize payout (weight: 1.0, 0.0)
- `"utilization"` - Maximize truck fill (weight: 0.0, 1.0)
- `"balanced"` - Balance both (weight: 0.5, 0.5)

**Custom Weights:**
- `revenue_weight`: 0.0 to 1.0
- `utilization_weight`: 0.0 to 1.0

**API Usage:**

```bash
# Pure revenue optimization (default)
curl -X POST http://localhost:8080/api/v1/load-optimizer/optimize \
  -d '{"truck": {...}, "orders": [...], "optimization_config": {"objective": "revenue"}}'

# Balanced optimization
curl -X POST http://localhost:8080/api/v1/load-optimizer/optimize \
  -d '{"truck": {...}, "orders": [...], "optimization_config": {"objective": "balanced"}}'

# Custom weights (80% revenue, 20% utilization)
curl -X POST http://localhost:8080/api/v1/load-optimizer/optimize \
  -d '{"truck": {...}, "orders": [...], "optimization_config": {"revenue_weight": 0.8, "utilization_weight": 0.2}}'

# Combine algorithm + objective
curl -X POST http://localhost:8080/api/v1/load-optimizer/optimize \
  -d '{"truck": {...}, "orders": [...], "optimization_config": {"algorithm": "backtracking", "objective": "balanced"}}'
```

**Validation:**
- Invalid objectives → 400 error
- Weights < 0 or > 1.0 → 400 error
- Invalid algorithm names → 400 error

---

## Security & Production Readiness

### Security Features
- Non-root Docker user
- Request size limiting
- Input validation
- Graceful shutdown (SIGTERM handling)
- Health checks

### Observability
- Structured logging
- Request/response logging
- Compute time tracking
- Health check endpoint

### Scalability
- Stateless (no session affinity needed)
- Horizontally scalable
- In-memory only (no DB bottleneck)
- Fast startup time

### Caching & Memoization Strategy

**Current Implementation:**
- **Precomputed Incompatibility Masks**: Before the DP loop, we compute a bitmask for each order showing which other orders it cannot be combined with. This converts O(N) compatibility checks to O(1) lookups during optimization.
- **State Memoization Ready**: The `DPOptimizer` includes a `cache map[int]domain.State` structure for memoizing intermediate states across multiple requests with similar order sets.
- **ClearCache() Method**: Available for cache management in production deployments.

**Thread Safety:**
- Current implementation is stateless and processes each request independently (thread-safe by design)
- Each request creates its own DP table and incompatibility masks
- No shared state between concurrent requests

**Future Production Enhancements:**
- **Distributed Caching**: Redis/Memcached for caching optimal solutions across service instances
- **Cache Key**: Hash of (truck capacity + sorted order IDs) for reusable results
- **Time-based Eviction**: Cache solutions with TTL based on order freshness
- **Cache Warming**: Pre-compute solutions for common truck/order combinations
- **Approximate Solutions**: For n > 25, use greedy/approximation with caching fallback

**Why This Matters:**
- Repeated queries with same orders benefit from cached results
- Distributed caching enables sub-millisecond response times
- Graceful degradation: Cache miss falls back to DP computation

## Edge Cases Handled

| Case | Handling |
|------|----------|
| Empty orders list | Returns empty selection with 0 payout |
| No feasible combination | Returns empty selection with metrics |
| Single order exceeds capacity | Filtered during preprocessing |
| Hazmat + non-hazmat mix | Enforces isolation constraint |
| Invalid dates | Returns 400 with clear error message |
| Conflicting time windows | Validates pickup <= delivery |
| Different routes | Only combines same origin-destination |
| Integer overflow | Uses int64 for all monetary calculations |

## Docker Details

### Multi-Stage Build

**Stage 1 (Builder):** Compile Go code  
**Stage 2 (Runtime):** Copy binary to Alpine

**Benefits:**
- Smaller image size
- Faster deployment
- Reduced attack surface
- Lower storage costs

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | HTTP server port |
| `LOG_LEVEL` | info | Logging verbosity |

### Resource Limits (docker-compose.yml)
- **CPU:** 2.0 cores max
- **Memory:** 512MB max

## Development

### Local Development

```bash
# Install Go dependencies
go mod download

# Build binary
go build -o server cmd/server/main.go

# Run server
./server
```
