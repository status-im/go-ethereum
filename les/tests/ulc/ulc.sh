#!/bin/bash
#
# Boot a LES/ULC network simulation using the HTTP API started by ulc.go.
#

set -e

main() {
  if ! which p2psim &>/dev/null; then
    fail "missing p2psim binary (you need to build cmd/p2psim and put it in \$PATH)"
  fi

  info "creating 5 LES nodes"
  for i in $(seq 1 5); do
    p2psim node create --name "$(les_node_name $i)"
    p2psim node start "$(les_node_name $i)"
  done

  info "creating one ULC node"
  p2psim node create --name "ulc"
  p2psim node start "ulc"

  info "connecting ULC node to all LES nodes"
  for i in $(seq 1 5); do
    p2psim node connect "ulc" "$(les_node_name $i)"
  done

  info "done"
}

les_node_name() {
  local num=$1
  echo "les$(printf '%02d' $num)"
}

info() {
  echo -e "\033[1;32m---> $(date +%H:%M:%S) ${@}\033[0m"
}

fail() {
  echo -e "\033[1;31mERROR: ${@}\033[0m" >&2
  exit 1
}

main "$@"
