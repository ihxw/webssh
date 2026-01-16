TermiScope Monitoring Agent
===========================

Supported OS:
- Linux (amd64, arm64)
- Windows (amd64)
- macOS (amd64, arm64)

Installation:
1. Upload/Copy the appropriate binary to your server/machine.
2. Make it executable (Linux/macOS): chmod +x termiscope-agent-*
3. Run it via TermiScope Dashboard "Deploy" button (Linux) or manually:

   Linux/macOS:
   ./termiscope-agent-[os]-[arch] -server http://YOUR_SERVER:3000 -secret YOUR_SECRET -id HOST_ID

   Windows (PowerShell/CMD):
   .\termiscope-agent-windows-amd64.exe -server http://YOUR_SERVER:3000 -secret YOUR_SECRET -id HOST_ID
