#!/bin/bash
exec >/proc/1/fd/1 2>&1
# check whether s6-svscan is running
check_s6_status() {
    if pgrep -x "s6-svscan" > /dev/null; then
        echo "s6-svscan is running."
        return 0  # normal
    else
        echo "s6-svscan is not running."
        return 1  # error
    fi
}

# startup s6-svscan
start_s6() {
    echo "Attempting to start s6-svscan..."
    export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH
    # startup command
    nohup s6-svscan /etc/s6 > /usr/local/builtin-script/copy/s6.log 2>&1 &
    echo "s6-svscan started successfully."
}

# main logic
if ! check_s6_status; then
    start_s6
else
    echo "No action needed."
fi
