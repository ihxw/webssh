# WebSSH
本项目代码全部由AI编写，后续bug也由AI修复.  
A modern, web-based SSH terminal designed for simplicity, security, and power.

## Features

- 🚀 **Fast & Responsive**: Built with Go, Vue 3, and xterm.js for high-performance terminal emulation.
- 📂 **SFTP Support**: Integrated file explorer for uploading, downloading, and managing files.
- 📹 **Session Recording**: Record your SSH sessions and replay them later using an integrated player.
- ⚡ **Quick Commands**: Reusable command templates for common tasks.
- 🔒 **Secure**: JWT-based authentication, one-time WebSocket tickets, and AES-encrypted host data.
- 💾 **Session Persistence**: Sessions stay active even when navigating through the dashboard.
- 🌗 **Theme Support**: Fully optimized for Light and Dark modes.

## Quick Start

### Using Docker (Recommended)

```bash
docker-compose up -d
```
Access the app at `http://localhost:9287`. Default credentials: `admin` / `admin123`.

### Local Development

1. **Backend**:
   ```bash
   go run cmd/server/main.go
   ```
2. **Frontend**:
   ```bash
   cd web
   npm install
   npm run dev
   ```

## Technology Stack

- **Backend**: Go (Gin, GORM, SQLite, x/crypto/ssh)
- **Frontend**: Vue 3, Ant Design Vue, Pinia, xterm.js
- **Database**: SQLite (built-in, pure Go implementation)

## Documentation

- [Deployment Guide](./DEPLOY.md)
- [Project Requirements](./需求.md)

## License

MIT
