#!/bin/bash

#
# Copyright (c) 2018 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#  http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#
# This script is useful to start the API server in a local environment, for
# development purposes. It assumes that an 'etcd' server is also started in the
# default address and port.
#

_output/local/bin/linux/amd64/autoheal \
apiserver \
--secure-port=8443 \
--etcd-servers=http://localhost:2379 \
--kubeconfig="${HOME}/.kube/config" \
--authorization-kubeconfig="${HOME}/.kube/config" \
--authentication-kubeconfig="${HOME}/.kube/config" \
 --logtostderr \
--v=8
