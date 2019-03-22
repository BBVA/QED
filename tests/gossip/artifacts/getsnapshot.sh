#!/bin/bash
# Copyright 2018-2019 Banco Bilbao Vizcaya Argentaria, S.A.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if [ -z "$1" ]; then
	echo usage:
	echo $0 qed_version
	exit -1
fi

raw=$(curl -s "http://localhost:8888/snapshot?v=$1")

history=$(echo $raw | jq -r '.Snapshot.HistoryDigest' | base64 -d | xxd -c64 -g64 | cut -f2 -d' ')
hyper=$(echo $raw | jq -r '.Snapshot.HyperDigest' | base64 -d | xxd -c64 -g64 | cut -f2 -d' ')

echo HyperDigest: $hyper
echo HistoryDigest: $history
