#!/bin/bash
# Raspberry Pi Diagnostic Script
# Run this on your Linux machine with Pi SD card inserted

set -e

echo "════════════════════════════════════════════════════════════════"
echo "Raspberry Pi Diagnostics & Setup"
echo "════════════════════════════════════════════════════════════════"
echo ""

# Check if SD card is mounted
echo "1. Checking for SD card..."
if ls /media/*/boot &> /dev/null || ls /media/*/bootfs &> /dev/null; then
    BOOT_MOUNT=$(ls -d /media/*/boot* 2>/dev/null | head -1)
    echo "✅ Found boot partition: $BOOT_MOUNT"
else
    echo "❌ No SD card boot partition found"
    echo ""
    echo "Please:"
    echo "  1. Insert Raspberry Pi SD card"
    echo "  2. Wait for auto-mount"
    echo "  3. Run this script again"
    echo ""
    echo "Or flash new image:"
    echo "  sudo apt install rpi-imager"
    echo "  rpi-imager"
    exit 1
fi

echo ""
echo "2. Checking SD card health..."
BOOT_DEV=$(df "$BOOT_MOUNT" | tail -1 | awk '{print $1}')
echo "Device: $BOOT_DEV"
sudo smartctl -H $BOOT_DEV 2>/dev/null || echo "⚠️  SMART data not available (normal for SD cards)"

echo ""
echo "3. Enabling SSH..."
if [ -f "$BOOT_MOUNT/ssh" ]; then
    echo "✅ SSH already enabled"
else
    sudo touch "$BOOT_MOUNT/ssh"
    echo "✅ SSH enabled"
fi

echo ""
echo "4. Checking HDMI config..."
if [ -f "$BOOT_MOUNT/config.txt" ]; then
    if grep -q "hdmi_force_hotplug" "$BOOT_MOUNT/config.txt"; then
        echo "✅ HDMI config already set"
    else
        echo "Adding HDMI force settings..."
        sudo tee -a "$BOOT_MOUNT/config.txt" > /dev/null << 'EOF'

# Force HDMI output (for troubleshooting blank screen)
hdmi_force_hotplug=1
hdmi_safe=1
config_hdmi_boost=4
EOF
        echo "✅ HDMI config updated"
    fi
fi

echo ""
echo "5. WiFi Configuration (Optional)"
read -p "Configure WiFi? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -p "WiFi SSID: " WIFI_SSID
    read -s -p "WiFi Password: " WIFI_PASS
    echo

    sudo tee "$BOOT_MOUNT/wpa_supplicant.conf" > /dev/null << EOF
country=US
ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
update_config=1

network={
    ssid="$WIFI_SSID"
    psk="$WIFI_PASS"
    key_mgmt=WPA-PSK
}
EOF
    echo "✅ WiFi configured"
fi

echo ""
echo "6. Creating first-boot script..."
sudo tee "$BOOT_MOUNT/firstboot.sh" > /dev/null << 'EOF'
#!/bin/bash
# This runs on first boot

# Update hostname
echo "ocx-pi" | sudo tee /etc/hostname
sudo sed -i 's/raspberrypi/ocx-pi/g' /etc/hosts

# Expand filesystem
sudo raspi-config --expand-rootfs

# Update packages
sudo apt-get update
sudo apt-get upgrade -y

# Install dependencies
sudo apt-get install -y \
    git \
    curl \
    build-essential \
    pkg-config \
    libssl-dev \
    cmake

# Install Go 1.21
cd /tmp
wget https://go.dev/dl/go1.21.5.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc

# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source ~/.cargo/env

# Clone OCX repo (if needed)
# git clone https://github.com/yourrepo/ocx-protocol ~/ocx-protocol

echo "✅ First boot setup complete!"
echo "Rebooting in 10 seconds..."
sleep 10
sudo reboot
EOF

sudo chmod +x "$BOOT_MOUNT/firstboot.sh"
echo "✅ First-boot script created"

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "✅ SD Card Setup Complete!"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "Next steps:"
echo "  1. Safely eject SD card: sync && sudo eject $BOOT_DEV"
echo "  2. Insert SD card into Raspberry Pi"
echo "  3. Connect HDMI to HDMI0 port (closest to USB-C power)"
echo "  4. Connect power (5V 3A recommended)"
echo "  5. Wait 2-3 minutes for first boot"
echo ""
echo "If screen still blank:"
echo "  - Check power LED (red) - should be solid ON"
echo "  - Check activity LED (green) - should blink during boot"
echo "  - Try different HDMI cable/monitor"
echo "  - Try HDMI1 port instead"
echo ""
echo "To connect via SSH (if WiFi configured):"
echo "  1. Find IP: sudo nmap -sn 192.168.1.0/24 | grep -B2 Raspberry"
echo "  2. SSH in: ssh pi@<IP_ADDRESS>"
echo "  3. Default password: raspberry (change immediately!)"
echo ""
echo "To run first-boot script manually:"
echo "  ssh pi@<IP_ADDRESS>"
echo "  bash /boot/firstboot.sh"
echo ""
