#!/bin/bash
set -euo pipefail
export MAN_DISABLE_CACHE=yes
export DEBIAN_FRONTEND=noninteractive

dpkg-divert --local --rename --add /var/lib/dpkg/info/man-db.triggers || true

USERNAME="{{ .Username }}"
PASSWORD="{{ .Password }}"
USER_HOME="/home/${USERNAME}"

echo "创建用户"

if ! id "${USERNAME}" &>/dev/null; then
    useradd -m -s /bin/bash "${USERNAME}"
fi

echo "${USERNAME}:${PASSWORD}" | chpasswd

mkdir -p "${USER_HOME}"
chown -R "${USERNAME}:${USERNAME}" "${USER_HOME}"
chmod 700 "${USER_HOME}"
