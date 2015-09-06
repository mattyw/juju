#!/bin/sh
juju bootstrap --upload-tools
juju set-env enable-os-refresh-update=false
juju set-env enable-os-upgrade=false
juju add-machine ssh:ubuntu@10.55.61.96 --debug
