#!/bin/bash

set -e

# Step 1: Copy directories to the specified locations
echo "Copying application files to /copy..."
cp -r /app/* /copy/
if [ $? -ne 0 ]; then
    echo "Error: Failed to copy application files. Exiting."
    exit 1
fi

# Step 2: Create /copy/bin and copy /usr/local/bin
echo "Creating /copy/bin and copying binaries..."
mkdir -p /copy/bin && cp -r /usr/local/bin /copy
if [ $? -ne 0 ]; then
    echo "Error: Failed to copy binaries. Exiting."
    exit 1
fi

# Step 3: Create /copy/lib and copy /usr/local/lib
echo "Creating /copy/lib and copying libraries..."
mkdir -p /copy/lib && cp -r /usr/local/lib /copy
if [ $? -ne 0 ]; then
    echo "Error: Failed to copy libraries. Exiting."
    exit 1
fi

echo "Copying pip and apt configurations..."
cp /app/pip.conf /pip-config-volume/pip.conf
if [ $? -ne 0 ]; then
    echo "Error: Failed to copy pip configuration. Exiting."
    exit 1
fi
cp /app/apt.sources.list /apt-resources/sources.list
if [ $? -ne 0 ]; then
    echo "Error: Failed to copy apt sources list. Exiting."
    exit 1
fi

# Copying custom pip and apt configurations...
if [ -f "/custom-pip-config-volume/pip.conf" ]; then
    echo "Copying custom pip configurations..."
    cp /custom-pip-config-volume/pip.conf /pip-config-volume/pip.conf
fi
if [ -f "/custom-apt-resources/apt.sources.list" ]; then
    echo "Copying custom apt resources configurations..."
    cp /custom-apt-resources/apt.sources.list /apt-resources/sources.list
fi

echo "All files copied successfully."
