## Install

Grab the latest copy of YAML from the [Releases page](https://github.com/drae/secretgen-controller/releases) and deploy it using kubectl.

Example:

```bash
kubectl apply -f https://github.com/drae/secretgen-controller/releases/latest/download/secretgen-controller.yaml
```

### Advanced

You can customize the deployment by using Kustomize with the base configurations provided in the repository:

```bash
git clone https://github.com/drae/secretgen-controller.git
cd secretgen-controller
kubectl apply -k config/kustomize/overlays/prod
```

For development purposes, you can use:

```bash
kubectl apply -k config/kustomize/overlays/dev
```

Next: [Walkthrough](walkthrough.md)
