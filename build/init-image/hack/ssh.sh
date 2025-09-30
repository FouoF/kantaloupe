#!/bin/bash
exec >/proc/1/fd/1 2>&1
set -e

# Function to check if sshd is installed
is_sshd_installed() {
    if command -v sshd &> /dev/null; then
        return 0
    else
        return 1
    fi
}

# Check the operating system type
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo "Unable to determine the operating system type."
    exit 1
fi

# Check if sshd is already installed
if is_sshd_installed; then
    echo "sshd is already installed."
else
    echo "Installing sshd..."

    # Install and start sshd based on the operating system type
    case "$OS" in
        ubuntu|debian)
            echo "Detected Ubuntu operating system."
            apt update
            apt install -y openssh-server
            ;;
        centos)
            echo "Detected CentOS operating system."
            sed -i 's|^mirrorlist=|#mirrorlist=|g' /etc/yum.repos.d/CentOS-*.repo
            sed -i 's|^#baseurl=http://mirror.centos.org|baseurl=http://vault.centos.org|g' /etc/yum.repos.d/CentOS-*.repo
            yum clean all
            yum makecache
            yum install -y openssh-server
            ;;
        *)
            echo "Unsupported operating system type: $OS"
            exit 1
            ;;
    esac
fi

# Ensure the /run/sshd directory exists
mkdir -p /run/sshd

# Enable root login and set default password
echo "Configuring sshd for root login..."
sed -i 's/^#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config

if [ -z "$ROOT_PASSWORD" ]; then
    echo "ERROR: ROOT_PASSWORD environment variable is not set."
    exit 1
fi

sed -i 's/^#AuthorizedKeysFile.*/AuthorizedKeysFile .ssh\/authorized_keys .ssh\/drun\/authorized_keys/' /etc/ssh/sshd_config

sed -i 's/^#StrictModes.*/StrictModes no/' /etc/ssh/sshd_config

echo "Setting root password..."
echo "root:$ROOT_PASSWORD" | chpasswd

echo "sshd setup complete."
