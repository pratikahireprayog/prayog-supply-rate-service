# Prayog Rate Card Service

A high-performance microservice for calculating shipping rates from multiple logistics partners using clean architecture principles and the factory pattern.

## 🚀 Features

- **Multi-Provider Support**: Integrates with both static and dynamic rate providers
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
│                   │  │Dynamic Providers    │ │   │
│                   │  │Static Providers     │ │   │
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

### Phase 2: Dynamic Rate Providers 🚧 NEXT
- [ ] Partner API integration framework
- [ ] Real-time rate calculation from external APIs
- [ ] Provider health monitoring and circuit breakers
- [ ] Rate caching and optimization
- [ ] Comprehensive testing suite

### Phase 3: Static Rate Integration 📋 PLANNED
- [ ] Database integration for static rates
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
   curl http://localhost:8080/health
   ```

The service will be available at `http://localhost:8080`

### Environment Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `HOST` | Server host | `0.0.0.0` |

## 📡 API Endpoints

### Health & Monitoring
- `GET /health` - Basic health check
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe
- `GET /metrics` - Service metrics

### Rate Calculation (Phase 1: Mock Implementation)
- `POST /api/v1/rates/calculate` - Calculate rates from all providers
- `POST /api/v1/rates/quote` - Get rate quote (alias for calculate)
- `POST /api/v1/rates/compare` - Compare rates from multiple providers
- `POST /api/v1/rates/best` - Get the best rate
- `POST /api/v1/rates/provider/:providerId` - Get rates from specific provider

### Provider Management
- `GET /api/v1/providers/health` - Get provider health status
- `POST /api/v1/providers/refresh` - Refresh provider configurations

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/rates/calculate \
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
│   │   └── providers/              # Rate provider implementations
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

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
go test ./test/...
```

## 📊 Monitoring

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
- **Q1 2024**: Phase 2 - Dynamic Rate Providers
- **Q2 2024**: Phase 3 - Static Rate Integration
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

**Status**: Phase 1 Complete ✅ | **Next**: Phase 2 - Dynamic Rate Providers 🚧