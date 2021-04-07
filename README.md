# promdump

promdump dumps Prometheus samples that fall within a time range, for transfer
to another Prometheus instance.

## Why This Tool

When debugging users' Kubernetes workloads, I find it invaluable to get access
to the Prometheus metrics on their clusters. And to reduce the amount of back
and forth (due to missing metrics, incorrect labels etc.), it makes sense to
ask the users to _"get me everything around the time of the incident"_.

The most common way to achieve this is to use commands like `kubectl exec` and
`kubectl cp` to compress and dump Prometheus' entire data directory. On
non-trivial clusters, the resulting compressed file can be very large. To
import the data into a test instance will require at least the same amount of
disk space.

promdump is a tool that can be used to dump Prometheus samples that fall within
a time range.

It is different from the `promtool tsdb dump` command as its output can be
copied over to another Prometheus instance. See this
[issue](https://github.com/prometheus/prometheus/issues/8281) for an in-depth
discussion on the limitation on the output of `promtool tsdb dump`.

And unlike the Promethues TSDB `snapshot` API, promdump doesn't require
Prometheus to be started with the `--web.enable-admin-api` option. Instead of
dumping the entire TSDB, promdump offers the flexibility to capture data that
falls within a particular time range.

## How It Works

The promdump CLI downloads the promdump-$(VERSION).tar.gz file from a public
Cloud Storage bucket to your local /tmp folder. The download will be skipped
if such a file already exists. The `-f` option can be used to force a
re-download.

Then the CLI uncompresses the archive file and uploads the promdump binary to
the targeted in-cluster Prometheus container, via the pod's `exec` subresource.

Within the Prometheus container, promdump queries the Prometheus TSDB using the
[`tsdb`](https://pkg.go.dev/github.com/prometheus/prometheus/tsdb) package. Data
blocks that fall within the specified time range are gathered and streamed to
stdout, which can be redirected to a compressed file on your local file system.

The `restore` subcommand can then be used to copy this compressed file to
another Prometheus instance.

## Getting Started


## LICENSE

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
these files except in compliance with the License. You may obtain a copy of the
License at:

```
http://www.apache.org/licenses/LICENSE-2.0
```

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.