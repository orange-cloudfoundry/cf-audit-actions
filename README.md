# cf-audit-actions

Audit cloud foundry objects and do action when found a potential vulnerability
For now only checking and disable ssh on app or space after a time limit has been made.

## Installation

```bash
$ bash -c "$(curl -fsSL https://raw.github.com/orange-cloudfoundry/cf-audit-actions/master/bin/install.sh)"
```

## Usage

```
Usage:
  cf-audit-actions [OPTIONS] <ssh | ssh-app>

Application Options:
  -a, --api=                 cf api endpoint
  -i, --client-id=           cf client id
  -s, --client-secret=       cf client id
  -u, --username=            cf username (if client-id can't bet set)'
  -p, --password=            cf password (if client-id can't bet set)
      --parallel=            how many parallel request can be made
  -k, --skip-ssl-validation  Skip ssl validation
  -v, --version              Show version

Help Options:
  -h, --help                 Show this help message

Available commands:
  ssh      Check if ssh is enabled in spaces and deactivate it if it reach the time limit
  ssh-app  Check if ssh is enabled in apps and deactivate it if it reach the time limit
```

## Concourse pipelines

Pipeline are provided in [/ci](/ci) folder:

- ssh audit and action on space:

```bash
fly set-pipeline \
    --pipeline ssh-space-audit-actions \
    --config ci/ssh/pipeline.yml \
    --load-vars-from ci/ssh/vars.yml
```

- ssh audit and action on app:
```bash
fly set-pipeline \
    --pipeline ssh-app-audit-actions \
    --config ci/ssh-app/pipeline.yml \
    --load-vars-from ci/ssh-app/vars.yml
```