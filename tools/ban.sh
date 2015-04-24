#!/usr/bin/env bash
#
# ban ip addresses manually
#
#cd $(dirname $0)
vals=""
for arg in $@ ; do
    vals="(\"$arg\",\"ban\", 0, 9000000),$vals"
done
sqlite3 livechan.db <<EOF
INSERT INTO Channels(ip) VALUES${vals:0:-1};
EOF
