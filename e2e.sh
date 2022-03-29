#!/usr/bin/env bash

BROKER_HOST="${BROKER_HOST:-localhost}"
BROKER_PORT="${BROKER_PORT:-1883}"

exec 3< <(mosquitto_sub -h "${BROKER_HOST}" -p "${BROKER_PORT}" -t "/gowon/output" -C 1 &)

PUB_COMMAND='{"module":"gowon","msg":".steam h","nick":"tester","dest":"#gowon","command":"steam","args":"h"}'
mosquitto_pub -h "${BROKER_HOST}" -p "${BROKER_PORT}" -t "/gowon/input" -m "${PUB_COMMAND}"

GOT_MSG=$(cat <&3)
EXPECTED_MSG='{"module":"steam","msg":"one of [s]et, [r]ecent or [a]chievements must be passed as a command","nick":"tester","dest":"#gowon","command":"steam","args":"h"}'

if [[ "${GOT_MSG}" == "${EXPECTED_MSG}" ]]; then
    echo "[0;32mEnd to end tests successful[0m"
else
    echo "[0;31mEnd to end tests unsuccessful[0m"
fi
