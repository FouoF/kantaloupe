#!/bin/bash

# Load environment variables
export $(cat /proc/1/environ | tr '\0' '\n' | xargs)
export PATH=$PATH:$HOME/.local/bin:/opt/conda/bin
# Load i18n translations
source "/usr/local/builtin-script/copy/i18n.sh"

# Load system information
source "/usr/local/builtin-script/copy/system_info.sh"

# Print translated system information
echo ""
echo "$TIPS_TITLE"
echo "+----------------------+-----------+---------------------------------------------------------------------------------------+"
echo "        $DIR_HEADER    |   $NAME_HEADER   |                                       $DESC_HEADER                                      "
echo "+----------------------+-----------+---------------------------------------------------------------------------------------+"
echo -e "  /                    |  $SYSTEM_DISK   | $SYSTEM_DISK_DESC  "
echo "  /root/data           |  $DATA_STORAGE  | $DATA_STORAGE_DESC  "
if [ -d "/root/public-model" ]; then
  echo "  /root/public-model   |  $MODEL_FILES   | $MODEL_FILES_DESC  "
fi
echo "+----------------------+----------+---------------------------------------------------------------------------------------+"
echo ""
echo "  -------- $SYSTEM_INFO --------"
echo ""
echo "  $OS_VERSION  : $os_version"
echo "  $KERNEL      : $kernel"
echo "  $IP_ADDR     : $ip_address"
echo "  $HOSTNAME    : $hostname"
echo ""
echo "  $CPU_MODEL   : $cpu_model"
echo "  $CPU_THREADS : $cpu_threads C"
echo "  $MEMORY      : ${memory_used} MB / ${memory_total} MB (${memory_percent}% $MEMORY_USED)"
echo "  $GPU         : $gpu_info"
echo "  $CUDA        : $cuda_version"
echo ""
echo "  ------ $FILESYSTEM_INFO ------"
echo ""
echo "  $MOUNTED: /root/data    $data_used / $data_total ($data_percent $MEMORY_USED)"
echo ""
