#!/bin/bash -ex

#
# Copyright (c) 2018 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# This script uses the auto-heal OpenShift template to create an instance with
# minimal configuration.

oc process \
--filename=template.yml \
--param=AWX_ADDRESS="https://my-awx.example.com/api" \
--param=AWX_USER="$(echo -n 'autoheal' | base64 --wrap=0)" \
--param=AWX_PASSWORD="$(echo -n 'redhat123' | base64 --wrap=0)" \
| \
oc create --filename=-

# Add a line like this if you have a `ca.crt` file containing the CA
# certificates needed to connect to the AWX server:
#
# --param=AWX_CA="$(base64 --wrap=0 ca.crt)" \
