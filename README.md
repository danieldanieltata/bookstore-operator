# bookstore-operator

Kubernetes operator for managing BookStores and Books (CRDs). BookStores get a namespace each; Books live in a store’s namespace and can optionally be created as a copy of another book.

## Key design decisions

Two controllers (Bookstore and Book) drive the flow. The diagram below summarizes it.

**Creation.** Create a Bookstore → Bookstore controller creates a Namespace with an ownerRef to the Bookstore (so the garbage collector can delete the namespace later). Create an original Book → Book controller sets its `status.referenceCount = 0`. Create a Book with `spec.copyOf` pointing at that original → Book controller fetches the original, bumps its `referenceCount`, and updates the copy's status (title, price, genre) from the original.

**Explicit cleanup (delete Bookstore).** A finalizer blocks deletion. The Bookstore controller: (1) lists Books in that stores namespace and deletes each one; (2) lists Books in all namespaces and deletes any where `spec.copyOf.namespace` is the store being removed (copies that came from this store). Only after the finalizer is removed does the garbage collector delete the Namespace, because of the ownerRef set at creation.

## Prerequisites

- Go 1.24+
- Docker 17.03+
- kubectl + a Kubernetes cluster (v1.11.3+)

## Running the project

I run it locally against whatever cluster my kubeconfig points at.

1. **Install the CRDs** (once per cluster):

   ```sh
   make install
   ```

2. **Start the controller** (no image build, runs on your machine).

   The webhook server needs TLS certs: the API server only talks to admission webhooks over HTTPS, so we have to serve our own cert when running locally. Generate them once, then point `make run` at the directory:

   **Generate certs** (e.g. in `/tmp/webhook-certs`):

   ```sh
   mkdir -p /tmp/webhook-certs
   openssl req -x509 -newkey rsa:4096 -keyout /tmp/webhook-certs/tls.key -out /tmp/webhook-certs/tls.crt -days 365 -nodes -subj "/CN=localhost"
   ```

   **Run** with that path:

   ```sh
    make run WEBHOOK_CERT_PATH=/tmp/webhook-certs
   ```

   The Makefile passes `WEBHOOK_CERT_PATH` into `--webhook-cert-path`; the manager expects `tls.crt` and `tls.key` in that directory. Leave it running in a terminal.

3. **Apply samples** (see below for the order).

### Samples I use for testing

Apply in this order (jerusalem’s book has a `copyOf` pointing at tel-aviv, so tel-aviv has to exist first):

1. **Tel-aviv** (store then book):

   ```sh
   kubectl apply -f config/samples/v1_bookstore_tel_aviv.yaml
   kubectl apply -f config/samples/v1_book_tel_aviv.yaml
   ```

2. **Jerusalem** (store then book):

   ```sh
   kubectl apply -f config/samples/v1_bookstore_jerusalem.yaml
   kubectl apply -f config/samples/v1_book_jerusalem.yaml
   ```

### Cleanup

Delete the sample resources, then remove the CRDs:

```sh
kubectl delete -f config/samples/v1_book_jerusalem.yaml -f config/samples/v1_bookstore_jerusalem.yaml \
  -f config/samples/v1_book_tel_aviv.yaml -f config/samples/v1_bookstore_tel_aviv.yaml
```

---

Run `make help` for other targets. More in the [Kubebuilder docs](https://book.kubebuilder.io/introduction.html).

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

