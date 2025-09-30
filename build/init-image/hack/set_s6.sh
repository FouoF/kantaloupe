#!/bin/bash
exec >/proc/1/fd/1 2>&1
set -e

echo "Starting setup..."

echo "init path: $PATH"

[ -f /etc/profile ] && . /etc/profile

echo "current path: $PATH"

WELCOME_SCRIPT="source /usr/local/builtin-script/copy/welcome.sh"

# Add welcome script to bashrc if not already present
echo "Adding welcome script to /etc/bash.bashrc..."
if ! grep -Fxq "$WELCOME_SCRIPT" /etc/bash.bashrc; then
    echo "$WELCOME_SCRIPT" >> /etc/bash.bashrc
    echo "Welcome script added to /etc/bash.bashrc."
else
    echo "Welcome script already present in /etc/bash.bashrc. Skipping."
fi

bash /usr/local/builtin-script/copy/set_system_lang.sh || true

SERVICES=("sshd" "jupyterlab" "code-server")
BASE_PATH="/etc/s6"
SOURCE_PATH="/usr/local/builtin-script/copy"

echo "Ensuring base directory exists..."
mkdir -p "$BASE_PATH"

echo "Copying bin and lib directories..."
cp -r "$SOURCE_PATH/bin" /usr/local
cp -r "$SOURCE_PATH/lib" /usr/local
echo "bin and lib directories copied successfully."

echo "Processing service directories based on environment variables..."
for SERVICE in "${SERVICES[@]}"; do
    ENV_VAR="ENABLE_${SERVICE^^}"
    ENV_VAR=$(echo "$ENV_VAR" | tr '-' '_')

    SRC_DIR="$SOURCE_PATH/s6/$SERVICE"
    DEST_DIR="$BASE_PATH/$SERVICE"

    echo "Checking $SERVICE..."
    if [ "$SERVICE" == "sshd" ] || [ "${!ENV_VAR}" == "true" ]; then
        if [ -d "$SRC_DIR" ]; then
            echo "Copying $SERVICE service directory from $SRC_DIR to $DEST_DIR..."
            cp -r "$SRC_DIR" "$DEST_DIR" || { echo "Error copying $SERVICE"; exit 1; }
        else
            echo "Warning: Source directory $SRC_DIR not found for $SERVICE."
        fi
    else
        echo "$SERVICE is not enabled. Skipping."
    fi
done

echo "Service directories processed successfully."

echo "Setting LD_LIBRARY_PATH environment variable..."
export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH
echo "LD_LIBRARY_PATH set to $LD_LIBRARY_PATH"

echo "Starting s6 service manager..."

nohup s6-svscan /etc/s6 > /usr/local/builtin-script/copy/s6.log 2>&1 &

echo "s6 service manager started successfully."
echo "Setup complete."
