# Kollektor

The Kollektor delivers as easy way to monitor current deployed versions of OSS applications and monitor new releases via Kubernetes `CRD`

### Operator features

* Check current deployed version of k8s images
    - Statefulset
    - DaemonSet
    - Deployment
    - Replicaset
    - Pod

* Monitor new releases on Github

* Monitor new helm chart releases on ArtifactHub

* Notifications:
    - [Slack](https://slack.com)

## Instalation

Github is having anti-DDoS protection, so if you are going to scrape them very ofthen, you need to specify `GITHUB_TOKEN`

```yaml
operator:
  config:
    scrape:
      githubTokenSecret: "" # Name of existing secret in operator's namespace (key: GITHUB_TOKEN)
```

### Slack

Operator supports slack integration, you need to specify in values

```yaml
operator:
  config:
    slack:
      enabled: true
      webhookUrlSecret: "" # Name of existing secret in operator's namespace (key: SLACK_WEBHOOK_URL)
```

## Usage

Create `CR` and deploy it into application namespace

For more resource you can find in [Examples](examples/)

```yaml
apiVersion: kollektor.pannoi/v1alpha1
kind: Kollektor
metadata:
  name: vault
spec:
  source:
    repo: https://github.com/hashicorp/vault # Github URL for project
    chartRepo: https://artifacthub.io/packages/helm/hashicorp/vault # ArtifactHub URL for project
  resource:
    type: statefulset # Kubernetes deployed resource 
    name: vault # Name of Resource
    # containerName: "" # Container name usually matches the resource name, if not please specify here
```