# Loggregator Bench

Component benchmarks for Loggregator.

How to run on bosh-lite:

```bash
bosh create-release
bosh -e lite upload-release
bosh -e lite -d loggregator-bench deploy --vars-store=./vars.yml manifests/loggregator-bench.yml
bosh -e lite -d loggregator-bench run-errand loggregator-bench
```
