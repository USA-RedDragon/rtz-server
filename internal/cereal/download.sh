#!/bin/bash

set -eu

# renovate: datasource=github-tags depName=commaai/openpilot
OPENPILOT_VERSION=v0.10.3
# renovate: sha: datasource=git-refs depName=opendbc packageName=commaai/opendbc branch=master
OPENDBC_SHA=15b8354e167d543392d0d46f2a3ac44e0b2f8a72

curl -fSsL https://raw.githubusercontent.com/commaai/opendbc/$OPENDBC_SHA/opendbc/car/car.capnp -o car.capnp
curl -fSsL https://raw.githubusercontent.com/commaai/openpilot/refs/tags/$OPENPILOT_VERSION/cereal/custom.capnp -o custom.capnp
curl -fSsL https://raw.githubusercontent.com/commaai/openpilot/refs/tags/$OPENPILOT_VERSION/cereal/log.capnp -o log.capnp
curl -fSsL https://raw.githubusercontent.com/commaai/openpilot/refs/tags/$OPENPILOT_VERSION/cereal/legacy.capnp -o legacy.capnp
sed -i 's#$Cxx.namespace("cereal");#$Go.package("cereal");\n$Go.import("internal/cereal");#' *.capnp
sed -i 's#using Cxx = import "./include/c++.capnp";#using Go = import "/go.capnp";#' *.capnp
