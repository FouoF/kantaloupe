#!/bin/bash
exec >/proc/1/fd/1 2>&1
set -e

# Function to check if a command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# Function to install Python
install_python() {
    echo "Installing Python..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if ! command_exists brew; then
            echo "Homebrew not found. Installing Homebrew..."
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        fi
        brew install python
    elif [[ -f /etc/centos-release ]]; then
        # CentOS
        yum install -y python3
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Ubuntu
        apt update && apt install -y python3
    else
        echo "Unsupported operating system: $OSTYPE"
        exit 1
    fi
}

# Function to install pip
install_pip() {
    echo "Installing pip..."
    if [[ "$OSTYPE" == "darwin"* || "$OSTYPE" == "linux-gnu"* ]]; then
        apt install -y python3-pip
    elif [[ -f /etc/centos-release ]]; then
        yum install -y python3-pip
    else
        echo "Unsupported operating system: $OSTYPE"
        exit 1
    fi
}

# Function to install JupyterLab
install_jupyterlab() {
    echo "Installing JupyterLab..."
    if pip3 help install | grep -q -- '--root-user-action'; then
        echo "pip3 version >= 23.1, using --root-user-action=ignore"
        pip3 install jupyterlab --root-user-action=ignore --break-system-packages
    else
        echo "pip3 version < 23.1, using --user"
        pip3 install jupyterlab --user
    fi
   
}


# Function to check if JupyterLab is installed
check_jupyterlab() {
    if pip3 show jupyterlab &> /dev/null; then
        echo "JupyterLab is already installed."
    else
        install_jupyterlab
    fi
}

# Main installation process
main() {
    echo "jupyterlab current path: $PATH"
    # Check for Python
    if command_exists python3; then
        echo "Python is already installed."
    else
        install_python
    fi

    # Check for pip
    if command_exists pip3; then
        echo "pip is already installed."
    else
        install_pip
    fi

    # Check for JupyterLab
    check_jupyterlab
}

# Start the installation process
main

echo "Installation complete."
