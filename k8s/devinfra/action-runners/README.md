# Managed by Helm

## Installation instructions

We use the [actions-runner-controller](https://github.com/actions/actions-runner-controller) Helm chart to deploy K8s runners for Github.

### Cert Manager

`actions-runner-controller` requires cert-manager. Ensure that cert-manager is already installed. If not, follow instructions to deploy cert manager [here](https://cert-manager.io/docs/installation/).

### Deploy Helm Chart

```
helm repo add actions-runner-controller https://actions-runner-controller.github.io/actions-runner-controller

helm upgrade --install --namespace actions-runner-system --create-namespace\
  --set=authSecret.create=true\
  --set=authSecret.github_token="REPLACE_YOUR_TOKEN_HERE"\
  --wait actions-runner-controller actions-runner-controller/actions-runner-controller
```

### Deploy the Runner

`kubectl apply -f runnerdeployment.yaml`
