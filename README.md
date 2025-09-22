# Prayog Rate Card Service

A high-performance microservice for calculating shipping rates from multiple logistics partners using clean architecture principles and the factory pattern.

## 🚀 Features

- **Multi-Provider Support**: Integrates with both pre-defined and real-time rate providers
- **Factory Pattern**: Extensible architecture for adding new rate providers
- **Clean Architecture**: Strict layer separation with dependency inversion
- **High Performance**: Built with Go 1.25.1 and Fiber v2 framework
- **Health Monitoring**: Comprehensive health checks for all providers
- **Rate Comparison**: Smart algorithms to find the best rates
- **Caching**: Intelligent caching strategy for better performance
- **Observability**: Built-in metrics, logging, and monitoring

## 🏗️ Architecture

The service follows **Clean Architecture** principles with strict versioning:

```
┌─────────────────────────────────────────────────┐
│                API Layer (Fiber)                │
├─────────────────────────────────────────────────┤
│              Business Logic Layer               │
│  ┌─────────────┐  ┌─────────────────────────┐   │
│  │Rate Service │  │   Factory Pattern       │   │
│  └─────────────┘  │  ┌─────────────────────┐ │   │
│                   │  │Real-time Providers  │ │   │
│                   │  │Pre-defined Providers│ │   │
│                   │  └─────────────────────┘ │   │
│                   └─────────────────────────────┘   │
├─────────────────────────────────────────────────┤
│                Shared Layer                     │
│  Models │ DTOs │ Interfaces │ Constants │ Utils │
└─────────────────────────────────────────────────┘
```

## 🛠️ Technology Stack

- **Language**: Go 1.25.1
- **Web Framework**: Fiber v2
- **Database**: PostgreSQL with GORM
- **Validation**: go-playground/validator
- **Testing**: Testify framework
- **Architecture**: Clean Architecture + Factory Pattern
- **API Documentation**: OpenAPI 3.0

## 🎯 Development Phases

### Phase 1: Project Structure ✅ COMPLETED
- [x] Clean architecture setup with strict versioning
- [x] Factory pattern implementation for rate providers
- [x] Base models, DTOs, and interfaces
- [x] Fiber HTTP server with middleware
- [x] Comprehensive error handling and validation
- [x] Health check and monitoring endpoints

### Phase 2: Real-time Rate Providers 🚧 NEXT
- [ ] Partner API integration framework
- [ ] Real-time rate calculation from external APIs
- [ ] Provider health monitoring and circuit breakers
- [ ] Rate caching and optimization
- [ ] Comprehensive testing suite

### Phase 3: Pre-defined Rate Integration 📋 PLANNED
- [ ] Database integration for pre-defined rates
- [ ] Rate management APIs
- [ ] Bulk rate import/export functionality
- [ ] Admin dashboard for rate management
- [ ] Advanced caching strategies

## 🚀 Quick Start

### Prerequisites

- Go 1.25.1 or higher
- Git

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/prayog/prayog-rate-service.git
   cd prayog-rate-service
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Run the service**:
   ```bash
   go run cmd/server/main.go
   ```

4. **Verify the service**:
   ```bash
   curl http://localhost:8080/rate/health
   ```

The service will be available at `http://localhost:8080`

## 🚀 Running the Service

### Development Mode

```bash
# Run directly with Go
go run cmd/server/main.go

# Run with custom port
PORT=9090 go run cmd/server/main.go

# Run with custom host and port
HOST=127.0.0.1 PORT=9090 go run cmd/server/main.go
```

### Production Mode

```bash
# Build the binary
go build -o bin/rate-service cmd/server/main.go

# Run the binary
./bin/rate-service

# Run in background
./bin/rate-service &

# Run with nohup for persistent background execution
nohup ./bin/rate-service > logs/service.log 2>&1 &
```

### Docker (Future)

```bash
# Build Docker image (planned for Phase 2)
docker build -t prayog-rate-service .

# Run with Docker (planned for Phase 2)
docker run -p 8080:8080 prayog-rate-service
```

### Service Management

```bash
# Check if service is running
curl http://localhost:8080/rate/health

# Stop the service (if running in background)
pkill -f rate-service

# View service logs (if using nohup)
tail -f logs/service.log

# Monitor service in real-time
watch -n 2 'curl -s http://localhost:8080/rate/health | jq'
```

### Environment Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `PORT` | Server port | `8080` | `PORT=9090` |
| `HOST` | Server host | `0.0.0.0` | `HOST=127.0.0.1` |
| `LOG_LEVEL` | Logging level | `info` | `LOG_LEVEL=debug` |
| `ENV` | Environment | `development` | `ENV=production` |

## 📡 API Endpoints

### Health & Monitoring
- `GET /rate/health` - Basic health check
- `GET /health/live` - Liveness probe (Kubernetes)
- `GET /health/ready` - Readiness probe (Kubernetes)
- `GET /rate/metrics` - Service metrics

### Rate Calculation (Phase 1: Mock Implementation)
- `POST /rate/v1/rates/calculate` - Calculate rates from all implementations
- `POST /rate/v1/rates/quote` - Get rate quote (alias for calculate)
- `POST /rate/v1/rates/compare` - Compare rates from multiple implementations
- `POST /rate/v1/rates/best` - Get the best rate
- `POST /rate/v1/rates/implementation/:implementationId` - Get rates from specific implementation

### Implementation Management
- `GET /rate/v1/implementations/health` - Get implementation health status
- `POST /rate/v1/implementations/refresh` - Refresh implementation configurations

### Example Request

```bash
curl -X POST http://localhost:8080/rate/v1/rates/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": "req-123",
    "customer_id": "cust-456",
    "origin_city": "Mumbai",
    "dest_city": "Delhi",
    "weight": 5.0,
    "distance": 1400.0,
    "service_type": "standard",
    "pickup_date": "2024-01-15T10:00:00Z",
    "delivery_date": "2024-01-17T18:00:00Z",
    "priority": "normal",
    "currency": "INR"
  }'
```

## 🏛️ Project Structure

```
prayog-rate-service/
├── cmd/
│   └── server/main.go              # Application entry point
├── internal/
│   ├── infrastructure/
│   │   └── api/http/               # HTTP server and routes
│   ├── services/v1/                # Business logic
│   │   ├── factory/                # Factory pattern implementation
│   │   └── implementations/        # Rate implementation classes
│   └── shared/
│       ├── constants/v1/           # Application constants
│       ├── dtos/v1/                # Data Transfer Objects
│       ├── interfaces/v1/          # Contract definitions
│       ├── models/v1/              # Database models
│       ├── repositories/v1/        # Data access layer
│       └── utils/v1/               # Helper functions
├── api_docs/                       # API documentation
├── test/                          # Integration tests
├── configs/                       # Configuration files
├── go.mod                         # Go module definition
└── README.md                      # This file
```

## 🧪 Testing

### Unit Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run integration tests
go test ./test/...
```

### API Testing

```bash
# Test basic health
curl http://localhost:8080/rate/health

# Test API info
curl http://localhost:8080/rate/

# Test rate calculation
curl -X POST http://localhost:8080/rate/v1/rates/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": "test-123",
    "customer_id": "cust-456", 
    "origin_city": "Mumbai",
    "dest_city": "Delhi",
    "weight": 5.0,
    "distance": 1400.0,
    "service_type": "standard",
    "pickup_date": "2025-01-15T10:00:00Z",
    "delivery_date": "2025-01-17T18:00:00Z",
    "priority": "normal",
    "currency": "INR",
    "source": "api"
  }'

# Test with specific implementation types
curl -X POST http://localhost:8080/rate/v1/rates/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": "test-types",
    "customer_id": "cust-456",
    "origin_city": "Mumbai", 
    "dest_city": "Delhi",
    "weight": 5.0,
    "distance": 1400.0,
    "service_type": "standard",
    "pickup_date": "2025-01-15T10:00:00Z",
    "delivery_date": "2025-01-17T18:00:00Z",
    "priority": "normal",
    "currency": "INR",
    "source": "api",
    "provider_types": ["pre_defined", "real_time"]
  }'

# Test implementation health
curl http://localhost:8080/rate/v1/implementations/health

# Test rate comparison
curl -X POST http://localhost:8080/rate/v1/rates/compare \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": "test-compare",
    "customer_id": "cust-456",
    "origin_city": "Mumbai",
    "dest_city": "Delhi", 
    "weight": 5.0,
    "distance": 1400.0,
    "service_type": "standard",
    "pickup_date": "2025-01-15T10:00:00Z",
    "delivery_date": "2025-01-17T18:00:00Z",
    "priority": "normal",
    "currency": "INR",
    "source": "api"
  }'
```

### Comprehensive Test Script

```bash
# Run the included test script
chmod +x test_service.sh
./test_service.sh
```

### Load Testing

```bash
# Install hey for load testing (if not installed)
go install github.com/rakyll/hey@latest

# Basic load test
hey -n 100 -c 10 http://localhost:8080/health

# Load test rate calculation endpoint
hey -n 50 -c 5 -m POST \
  -H "Content-Type: application/json" \
  -d '{"request_id":"load-test","customer_id":"cust-456","origin_city":"Mumbai","dest_city":"Delhi","weight":5.0,"distance":1400.0,"service_type":"standard","pickup_date":"2025-01-15T10:00:00Z","delivery_date":"2025-01-17T18:00:00Z","priority":"normal","currency":"INR","source":"api"}' \
  http://localhost:8080/rate/v1/rates/calculate
```

### Quick Commands Reference

```bash
# 🚀 Start Service
go run cmd/server/main.go                    # Development mode
./bin/rate-service                           # Production mode  
./bin/rate-service &                         # Background mode

# 🧪 Test Service
curl http://localhost:8080/health            # Health check
curl http://localhost:8080/rate/             # API info
./test_service.sh                            # Comprehensive test

# 🛑 Stop Service
pkill -f rate-service                        # Stop background service
Ctrl+C                                       # Stop foreground service

# 🔍 Monitor Service
tail -f logs/service.log                     # View logs
watch curl -s http://localhost:8080/health   # Monitor health
```

## 📊 Monitoring & Debugging

### Health Monitoring

```bash
# Basic health check
curl http://localhost:8080/rate/health

# Deep health check (includes implementation status)
curl http://localhost:8080/rate/health?deep=true

# Implementation health status
curl http://localhost:8080/rate/v1/implementations/health

# Service metrics
curl http://localhost:8080/metrics
```

### Logging & Debugging

```bash
# View real-time logs (if running in foreground)
# Logs will show in the terminal

# View background service logs
tail -f logs/service.log

# Monitor HTTP requests
curl -s http://localhost:8080/rate/v1/rates/calculate \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: debug-123" \
  -d '{"request_id":"debug-test","customer_id":"cust-456","origin_city":"Mumbai","dest_city":"Delhi","weight":5.0,"distance":1400.0,"service_type":"standard","pickup_date":"2025-01-15T10:00:00Z","delivery_date":"2025-01-17T18:00:00Z","priority":"normal","currency":"INR","source":"api"}'

# Check service status with detailed response
curl -s http://localhost:8080/health | jq
```

### Performance Monitoring

```bash
# Monitor response times
time curl -s http://localhost:8080/health

# Monitor memory usage (macOS)
ps aux | grep rate-service

# Monitor CPU usage (macOS) 
top -pid $(pgrep rate-service)
```

The service provides comprehensive monitoring:

- **Health Checks**: Multiple endpoints for different health aspects
- **Metrics**: Performance and business metrics via `/metrics`
- **Logging**: Structured logging with request tracing
- **Request Tracking**: Unique request IDs for debugging

## 🔧 Configuration

The service supports configuration through:

1. **Environment Variables** (highest priority)
2. **Configuration Files** (planned for Phase 2)
3. **Command Line Arguments** (planned for Phase 2)
4. **Default Values** (lowest priority)

## 🤝 Contributing

We welcome contributions! Please see our contributing guidelines:

1. **Architecture**: Follow clean architecture principles
2. **Code Style**: Use `gofmt` and `golint`
3. **Testing**: Maintain test coverage above 80%
4. **Documentation**: Update API docs for any endpoint changes
5. **Versioning**: All changes must be backward compatible

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes following the coding standards
4. Add tests for new functionality
5. Run `go test ./...` to ensure all tests pass
6. Submit a pull request

## 📋 Roadmap

- **Q1 2024**: Phase 1 - Project Structure ✅
- **Q1 2024**: Phase 2 - Real-time Rate Providers
- **Q2 2024**: Phase 3 - Pre-defined Rate Integration
- **Q2 2024**: Performance Optimization
- **Q3 2024**: Advanced Features (ML-based recommendations)
- **Q3 2024**: Multi-region deployment support

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

For support and questions:

- **Issues**: [GitHub Issues](https://github.com/prayog/prayog-rate-service/issues)
- **Documentation**: [API Documentation](./api_docs/)
- **Email**: support@prayog.com

## 🙏 Acknowledgments

Built with ❤️ by the Prayog team using:

- [Fiber](https://gofiber.io/) - Express-inspired web framework
- [GORM](https://gorm.io/) - Object Relational Mapping library
- [Validator](https://github.com/go-playground/validator) - Struct validation
- [UUID](https://github.com/google/uuid) - UUID generation

---

**Status**: Phase 1 Complete ✅ | **Next**: Phase 2 - Real-time Rate Providers 🚧