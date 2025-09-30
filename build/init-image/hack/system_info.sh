#!/bin/bash

# Get current system information
current_timestamp=$(date +"%Y-%m-%d %H:%M:%S")
os_version=$(grep PRETTY_NAME /etc/os-release | cut -d'"' -f2)
kernel=$(uname -r)
hostname=$(hostname)
ip_address=$(hostname -I | awk '{print $1}')

# CPU Information
cpu_model=$(lscpu | grep "Model name:" | sed 's/.*Model name:\s*//')
cpu_threads=$(grep -c '^processor' /proc/cpuinfo)

# Memory Usage
memory_total=$(free -m | awk '/Mem:/ {print $2}')
memory_used=$(free -m | awk '/Mem:/ {print $3}')
memory_percent=$(awk "BEGIN {printf \"%.2f\", ($memory_used/$memory_total)*100}")

gpu_model=""
gpu_count=""
cuda_version=""

# Check for NVIDIA GPU first
if command -v nvidia-smi > /dev/null 2>&1; then
  gpu_model=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>&1)
  ret_code=$?
  gpu_model=$(echo "$gpu_model" | head -n 1)
  if [[ $ret_code -eq 0 && -n "$gpu_model" ]]; then
    gpu_count=$(nvidia-smi --query-gpu=name --format=csv,noheader | wc -l)
  fi
# If no NVIDIA GPU, check for Biren GPU
elif command -v brsmi > /dev/null 2>&1; then
  gpu_model=$(brsmi gpu --query-gpu=name --format=csv,noheader 2>&1)
  ret_code=$?
  gpu_model=$(echo "$gpu_model" | head -n 1)
  if [[ $ret_code -eq 0 && -n "$gpu_model" ]]; then
    gpu_count=$(brsmi gpu --query-gpu=name --format=csv,noheader | wc -l)
  fi
# Check for Enflame GCU (S60/T10/T20 etc.)
elif command -v efsmi > /dev/null 2>&1; then
  gpu_model=$(efsmi --d DRIVER 2>&1 | awk '/^[[:space:]]*\|[[:space:]]+[0-9]+[[:space:]]+[A-Za-z0-9]+/{print $3; exit}')
  gpu_count=$(efsmi -L 2>&1 | grep -cE '^[[:space:]]*[0-9]+[[:space:]]+[A-Z0-9]+')
else
  gpu_model="NO GPU detected"
fi

if [[ -z "$gpu_count" ]]; then
  gpu_info=$gpu_model
else
  gpu_info="$gpu_model * $gpu_count"
fi

# CUDA Version
if command -v nvcc > /dev/null 2>&1; then
  cuda_version=$(nvcc --version | awk '/release/ {print $5}' | sed 's/,//;s/V//')
else command -v nvidia-smi > /dev/null 2>&1
  cuda_version=$(nvidia-smi 2>/dev/null | awk '/CUDA Version:/ {print $9}')
fi

# If no CUDA version detected
if [ -z "$cuda_version" ]; then
  cuda_version="NO CUDA detected"
fi

# Disk Usage for /root/data
if df -h /root/data > /dev/null 2>&1; then
  data_total=$(df -h /root/data | awk 'NR==2 {print $2}')
  data_used=$(df -h /root/data | awk 'NR==2 {print $3}')
  data_percent=$(df -h /root/data | awk 'NR==2 {print $5}')
else
  data_total="Not Mounted"
  data_used="Not Mounted"
  data_percent="Not Mounted"
fi
