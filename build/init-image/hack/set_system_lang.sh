#!/bin/bash
set -e  # Exit immediately if a command exits with a non-zero status

# Ensure root privileges
if [ "$(id -u)" -ne 0 ]; then
    echo "Error: Please run this script as root." >&2
    exit 1
fi

# Set target language, default to zh_CN.UTF-8
TARGET_LANG="${TARGET_LANG:-zh_CN.UTF-8}"

echo "🌍 Target language: $TARGET_LANG"

# Detect system type
if [ -f "/etc/os-release" ]; then
    . /etc/os-release
    OS_FAMILY=$ID  # Extract distro identifier (e.g., ubuntu, debian, arch)
else
    echo "⚠️ Unable to detect system type, compatibility issues may occur." >&2
    OS_FAMILY="unknown"
fi

# Check if `locale` command exists
if ! command -v locale-gen &>/dev/null; then
    echo "🔍 locale command not found, attempting to install locales..."
    if [[ "$OS_FAMILY" == "debian" || "$OS_FAMILY" == "ubuntu" ]]; then
        apt-get update
        apt-get install -y locales
    elif [[ "$OS_FAMILY" == "arch" ]]; then
        pacman -Sy --noconfirm glibc
    else
        echo "❌ Unsupported system, please install locales manually." >&2
        exit 1
    fi
fi

# Check if locale already exists
if locale -a | grep -iq "^${TARGET_LANG}$"; then
    echo "✅ Locale ${TARGET_LANG} already exists, configuring system..."
else
    echo "🌱 Locale ${TARGET_LANG} not found, installing now..."

    # Debian/Ubuntu handling
    if [[ "$OS_FAMILY" == "debian" || "$OS_FAMILY" == "ubuntu" ]]; then
        locale-gen ${TARGET_LANG}
    else
        echo "❌ Unsupported system, please configure locale manually." >&2
        exit 1
    fi
fi

# Update .profile
if [ -f ~/.bashrc ]; then
    # Remove existing LANG, LC_ALL, LC_TIME settings if file exists
    sed -i "/^export LANG=/d" ~/.bashrc
    sed -i "/^export LC_ALL=/d" ~/.bashrc
    sed -i "/^export LC_TIME=/d" ~/.bashrc
else
    # Create file if it doesn't exist
    touch ~/.bashrc
fi

# Add new configurations
echo "export LANG=${TARGET_LANG}" >> ~/.bashrc
echo "export LC_ALL=${TARGET_LANG}" >> ~/.bashrc
echo "export LC_TIME=${TARGET_LANG}" >> ~/.bashrc

source ~/.bashrc

echo -e "\n✅ Locale configuration complete! Please log out and log back in or restart your terminal for changes to take full effect."