#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"
SSH_ACCESS_PUBLIC_KEY="{{ .SSHAccessPublicKey }}"
SSH_GOMC_PUBLIC_KEY="{{ .SSHGomcPublicKey }}"

echo "配置 SSH authorized_keys"

SSH_DIR="${USER_HOME}/.ssh"
AUTHORIZED_KEYS="${SSH_DIR}/authorized_keys"

mkdir -p "${SSH_DIR}"
chmod 700 "${SSH_DIR}"

if ! grep -qxF "${SSH_ACCESS_PUBLIC_KEY}" "${AUTHORIZED_KEYS}" 2>/dev/null; then
    echo "${SSH_ACCESS_PUBLIC_KEY}" >> "${AUTHORIZED_KEYS}"
fi

if ! grep -qxF "${SSH_GOMC_PUBLIC_KEY}" "${AUTHORIZED_KEYS}" 2>/dev/null; then
    echo "${SSH_GOMC_PUBLIC_KEY}" >> "${AUTHORIZED_KEYS}"
fi

chown -R "${USERNAME}:${USERNAME}" "${SSH_DIR}"
chmod 600 "${AUTHORIZED_KEYS}"
