# SecretGen Controller

SecretGen Controller provides support for generated Kubernetes Secrets.

## Features

- [Secret Template](secret-template.md): Template a Secret from multiple sources using JSONPath
- [Secret Template Field Usage](secret-template-field.md): Detailed usage of SecretTemplate fields and expressions

## Installation

SecretGen Controller can be deployed in a variety of ways:

- [kapp](https://carvel.dev/kapp/): `kapp deploy -a sgc -f https://github.com/drae/secretgen-controller/releases/latest/download/release.yml`
- [kubectl](https://github.com/kubernetes/kubectl): `kubectl apply -f https://github.com/drae/secretgen-controller/releases/latest/download/release.yml`

## Usage

Please see feature usage documentation linked above.
