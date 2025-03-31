## Install

Grab the latest copy of YAML from the [Releases page](https://github.com/drae/templatedsecret-controller/releases) and deploy it using kubectl.

Example:

```bash
kubectl apply -f https://github.com/drae/templatedsecretsecret-controller/releases/latest/dowtemplatedsecretplatedsecret-controller.yaml
```

### Advanced

You can customize the deployment by using Kustomize with the base configurations provided in the repository:

```bash
git clone https://github.com/drae/templatedsecretsecret-controller.git
cd templatedsecretsecret-controller
kubectl apply -k config/kustomize/overlays/prod
```

For development purposes, you can use:

```bash
kubectl apply -k config/kustomize/overlays/dev
```

Next: [Walkthrough](walkthrough.md)
