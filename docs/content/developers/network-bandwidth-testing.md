---
search:
  boost: .1
---

# Network Bandwidth Testing Environments

!!!tip "Related Testing"
    For testing HTTP proxies, see [HTTP Proxy Test Environments](http-proxy-test-environments.md).

    For testing SSL interception scenarios, see [Network Test Environments: Packet-Inspection VPN Simulation](network-test-environments.md).

This guide demonstrates how to simulate slow network connections for testing DDEV behavior under bandwidth-constrained conditions.
This is useful for testing how DDEV performs in environments with limited internet connectivity, such as rural areas, mobile connections, or congested networks.

## Setup Overview

The setup involves:

- Identifying your network interface
- Installing network bandwidth limiting tools
- Applying bandwidth restrictions
- Testing DDEV operations under constrained conditions
- Removing bandwidth limitations when testing is complete

## Step-by-Step Instructions

### 1. Identify Your Network Interface

First, you need to identify which network interface you want to limit:

```bash
# List all network interfaces
ip link show

# Or use the older ifconfig command (if installed)
ifconfig -a

# Look for interfaces like:
# - eth0, enp7s0f1 (Ethernet)
# - wlan0, wlp3s0 (Wi-Fi)
# - docker0 (Docker bridge - avoid limiting this)
```

Common interface naming patterns:

- **Ethernet**: `eth0`, `enp7s0f1`, `eno1`
- **Wi-Fi**: `wlan0`, `wlp3s0`, `wlo1`
- **USB/Mobile**: `usb0`, `wwan0`

### 2. Install Bandwidth Limiting Tools

#### Wondershaper

[Wondershaper](https://github.com/magnific0/wondershaper) is a simple bandwidth limiting tool that's easy to use on Linux:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install wondershaper

# CentOS/RHEL/Fedora
sudo dnf install wondershaper

# Arch Linux
yay -S wondershaper-git
```

### 3. Apply Bandwidth Limitations

#### Using Wondershaper

Apply download and upload limits (values in kilobits per second):

```bash
# Basic usage: wondershaper -a <interface> -d <download_kbps> -u <upload_kbps>

# Simulate very slow connection (1 Mbps download)
sudo wondershaper -a enp7s0f1 -d 1024

# Simulate slow broadband (5 Mbps download)
sudo wondershaper -a enp7s0f1 -d 5120 -u 1024

# Simulate mobile connection (10 Mbps download)
sudo wondershaper -a enp7s0f1 -d 10240 -u 2048

# Remove all limitations
sudo wondershaper -c -a enp7s0f1
```

Replace `enp7s0f1` with your actual network interface name from step 1.

!!!warning "Remember to Remove Limits"
    Remember to remove bandwidth limitations after testing with `sudo wondershaper -c -a <interface>` to restore normal network performance. Consider adding this command to your testing scripts as cleanup.

### 4. Test DDEV Operations

With bandwidth limitations in place, test various DDEV operations:

```bash
# Cleanup before testing
ddev poweroff
ddev delete --omit-snapshot
docker builder prune
docker volume rm ddev-global-cache

# Test fresh DDEV startup
ddev config --webimage-extra-packages=htop --nodejs-version=18
DDEV_VERBOSE=true ddev start
# Open the logs to watch in another terminal:
ddev logs -f

# Force download of docker-compose binary
rm -f ~/.ddev/bin/docker-compose && DDEV_DEBUG=true ddev version

# Test composer operations
ddev composer install
```
