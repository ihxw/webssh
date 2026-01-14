# WebSSH

A modern web-based SSH terminal application built with Golang and Vue 3.

## Features

- 🔐 **Secure Authentication**: JWT-based authentication with bcrypt password hashing
- 🖥️ **Web Terminal**: Full-featured terminal emulator using xterm.js
- 📁 **SSH Host Management**: Save and manage SSH connection configurations
- 🔑 **Multiple Auth Methods**: Support for password and SSH key authentication
- 🔒 **Encrypted Storage**: AES-256-GCM encryption for sensitive credentials
- 👥 **User Management**: Admin panel for user administration
- 📊 **Connection History**: Track and audit SSH connections
- 🎨 **Modern UI**: Clean, compact interface with Ant Design Vue
- 🌓 **Dark Theme**: Eye-friendly dark theme for terminal work

## Tech Stack

### Backend
- **Language**: Golang
- **Framework**: Gin
- **Database**: SQLite with GORM
- **WebSocket**: gorilla/websocket
- **SSH**: golang.org/x/crypto/ssh

### Frontend
- **Framework**: Vue 3
- **UI Library**: Ant Design Vue
- **Terminal**: xterm.js
- **Build Tool**: Vite
- **State Management**: Pinia

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- Git

### Installation

1. Clone the repository:
```bash
git clone https://github.com/ihxw/webssh.git
cd webssh
```

2. Install dependencies:
```bash
# Install Go dependencies
go mod download

# Install frontend dependencies
cd web
npm install
cd ..
```

3. Set up environment variables:
```bash
# Generate a random JWT secret (32 characters)
$env:WEBSSH_JWT_SECRET = "your-super-secret-jwt-key-here-32chars"

# Generate a random encryption key (exactly 32 bytes)
$env:WEBSSH_ENCRYPTION_KEY = "12345678901234567890123456789012"
```

4. Run the development server:
```bash
go run cmd/server/main.go
```

5. In a separate terminal, run the frontend dev server:
```bash
cd web
npm run dev
```

6. Open your browser and navigate to `http://localhost:5173`

### Default Credentials

- **Username**: `admin`
- **Password**: `admin123`

**⚠️ IMPORTANT**: Change the default password immediately after first login!

## Configuration

Configuration can be set via `configs/config.yaml` or environment variables:

```yaml
server:
  port: 8080
  mode: debug  # or release

database:
  path: ./data/webssh.db

security:
  jwt_secret: ""  # Set via WEBSSH_JWT_SECRET
  encryption_key: ""  # Set via WEBSSH_ENCRYPTION_KEY (32 bytes)

ssh:
  timeout: 30s
  max_connections_per_user: 10

log:
  level: info
  file: ./logs/app.log
```

### Environment Variables

- `WEBSSH_PORT`: Server port (default: 8080)
- `WEBSSH_DB_PATH`: Database file path (default: ./data/webssh.db)
- `WEBSSH_JWT_SECRET`: JWT signing secret (required)
- `WEBSSH_ENCRYPTION_KEY`: AES-256 encryption key, exactly 32 bytes (required)

## Building for Production

1. Build the frontend:
```bash
cd web
npm run build
cd ..
```

2. Build the backend:
```bash
go build -o bin/webssh.exe cmd/server/main.go
```

3. Run the production binary:
```bash
# Set environment variables
$env:WEBSSH_JWT_SECRET = "your-secret"
$env:WEBSSH_ENCRYPTION_KEY = "your-32-byte-key-exactly-32bytes"

# Run
./bin/webssh.exe
```

## Docker Deployment

```bash
# Build Docker image
docker build -t webssh:latest .

# Run container
docker run -d \
  -p 8080:8080 \
  -e WEBSSH_JWT_SECRET=your-secret \
  -e WEBSSH_ENCRYPTION_KEY=your-32-byte-key-exactly-32bytes \
  -v $(pwd)/data:/app/data \
  webssh:latest
```

## API Documentation

### Authentication

#### POST /api/auth/login
Login with username/email and password.

**Request:**
```json
{
  "username": "admin",
  "password": "admin123",
  "remember": false
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@localhost",
      "role": "admin"
    }
  }
}
```

### SSH Hosts

#### GET /api/ssh-hosts
Get list of SSH hosts for the current user.

#### POST /api/ssh-hosts
Create a new SSH host configuration.

**Request:**
```json
{
  "name": "My Server",
  "host": "192.168.1.100",
  "port": 22,
  "username": "root",
  "auth_type": "password",
  "password": "secret",
  "group_name": "Production",
  "description": "Main production server"
}
```

### WebSocket SSH Connection

#### WebSocket /api/ws/ssh/:hostId
Establish SSH connection via WebSocket.

**Messages:**
- Input: `{"type": "input", "data": "ls -la\n"}`
- Resize: `{"type": "resize", "data": {"rows": 24, "cols": 80}}`

## Security Considerations

1. **Always use HTTPS in production** to protect credentials in transit
2. **Change default admin password** immediately
3. **Use strong JWT secret** (at least 32 random characters)
4. **Use proper encryption key** (exactly 32 random bytes for AES-256)
5. **Regularly update dependencies** to patch security vulnerabilities
6. **Implement rate limiting** for login endpoints in production
7. **Enable host key verification** for SSH connections (currently disabled)

## Development

### Project Structure

```
webssh/
├── cmd/server/          # Application entry point
├── internal/            # Internal packages
│   ├── config/          # Configuration management
│   ├── database/        # Database initialization
│   ├── models/          # Data models
│   ├── middleware/      # HTTP middleware
│   ├── handlers/        # HTTP handlers
│   ├── ssh/             # SSH client wrapper
│   └── utils/           # Utility functions
├── web/                 # Frontend Vue application
├── configs/             # Configuration files
├── data/                # Database files (gitignored)
├── logs/                # Log files (gitignored)
└── bin/                 # Compiled binaries (gitignored)
```

### Running Tests

```bash
# Backend tests
go test ./...

# Frontend tests
cd web
npm run test
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

## Acknowledgments

- [xterm.js](https://xtermjs.org/) - Terminal emulator
- [Gin](https://gin-gonic.com/) - Web framework
- [Ant Design Vue](https://antdv.com/) - UI components
- [GORM](https://gorm.io/) - ORM library
