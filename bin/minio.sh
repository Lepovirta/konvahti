#!/usr/bin/env sh
#
# Run MinIO for integration testing purposes
#
set -eu

UIDGID="$(id -u):$(id -g)"
MINIO_USER="${MINIO_USER:-"AKIAIOSFODNN7EXAMPLE"}"
MINIO_PASSWORD="${MINIO_PASSWORD:-"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}"

start_minio() {
  if ! docker volume ls -q | grep -q minio; then
    docker volume create minio
    docker run --rm -v minio:/miniodata \
      busybox /bin/sh -c \
      "touch /miniodata/.initialized && chown -R ${UIDGID} /miniodata"
  fi

  docker run \
    --rm \
    -d \
    --name minio \
    -p 9000:9000 \
    -p 9001:9001 \
    --user "${UIDGID}" \
    -e "MINIO_ROOT_USER=${MINIO_USER}" \
    -e "MINIO_ROOT_PASSWORD=${MINIO_PASSWORD}" \
    -v minio:/data \
    quay.io/minio/minio server /data --console-address ":9001"
}

case "${1:-}" in
    "stop") docker stop minio ;;
    "start") start_minio ;;
    "*") echo "usage: $0 start|stop" >&2 ;;
esac
