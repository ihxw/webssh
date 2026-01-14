# WebSSH Setup Script
# This script helps you set up the required environment variables

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "WebSSH Environment Setup" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Generate JWT Secret (32 characters)
$jwtSecret = -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 32 | ForEach-Object {[char]$_})

# Generate Encryption Key (exactly 32 bytes)
$encryptionKey = -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 32 | ForEach-Object {[char]$_})

Write-Host "Generated Secrets:" -ForegroundColor Green
Write-Host ""
Write-Host "JWT Secret:" -ForegroundColor Yellow
Write-Host $jwtSecret
Write-Host ""
Write-Host "Encryption Key (32 bytes):" -ForegroundColor Yellow
Write-Host $encryptionKey
Write-Host ""
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Setting Environment Variables..." -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Set environment variables for current session
$env:WEBSSH_JWT_SECRET = $jwtSecret
$env:WEBSSH_ENCRYPTION_KEY = $encryptionKey

Write-Host "Environment variables set for current session!" -ForegroundColor Green
Write-Host ""
Write-Host "To make these permanent, add them to your system environment variables:" -ForegroundColor Yellow
Write-Host ""
Write-Host "WEBSSH_JWT_SECRET=$jwtSecret" -ForegroundColor White
Write-Host "WEBSSH_ENCRYPTION_KEY=$encryptionKey" -ForegroundColor White
Write-Host ""
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. Install frontend dependencies:" -ForegroundColor White
Write-Host "   cd web" -ForegroundColor Gray
Write-Host "   npm install" -ForegroundColor Gray
Write-Host ""
Write-Host "2. Start the backend server:" -ForegroundColor White
Write-Host "   go run cmd/server/main.go" -ForegroundColor Gray
Write-Host ""
Write-Host "3. In another terminal, start the frontend dev server:" -ForegroundColor White
Write-Host "   cd web" -ForegroundColor Gray
Write-Host "   npm run dev" -ForegroundColor Gray
Write-Host ""
Write-Host "4. Open your browser and navigate to:" -ForegroundColor White
Write-Host "   http://localhost:5173" -ForegroundColor Gray
Write-Host ""
Write-Host "5. Login with default credentials:" -ForegroundColor White
Write-Host "   Username: admin" -ForegroundColor Gray
Write-Host "   Password: admin123" -ForegroundColor Gray
Write-Host ""
Write-Host "⚠️  IMPORTANT: Change the default password after first login!" -ForegroundColor Red
Write-Host ""
