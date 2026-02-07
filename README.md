# bookstore-operator

Kubernetes operator for managing BookStores and Books (CRDs). BookStores get a namespace each. Books live in a stores namespace and can optionally be created as a copy of another book.

## Key design decisions

Two controllers (Bookstore and Book) drive the flow. The diagram below summarizes it.
<img width="1157" height="1040" alt="image" src="https://github.com/user-attachments/assets/ce4cdf43-e2e7-4ca1-9f62-24ad79c1b35c" />

**Creation.** Create a Bookstore → Bookstore controller creates a Namespace with an ownerRef to the Bookstore (so the garbage collector can delete the namespace later). Create an original Book → Book controller sets its `status.referenceCount = 0`. Create a Book with `spec.copyOf` pointing at that original → Book controller fetches the original, bumps its `referenceCount`, and updates the copy's status (title, price, genre) from the original.

**Explicit cleanup (delete Bookstore).** A finalizer blocks deletion. The Bookstore controller:
(1) lists Books in that stores namespace and deletes each one
(2) lists Books in all namespaces and deletes any where `spec.copyOf.namespace` is the store being removed (copies that came from this store). Only after the finalizer is removed does the garbage collector delete the Namespace, because of the ownerRef set at creation.

**Delete in finalizer, not ownerRef for in-namespace Books.** With owner references, in-namespace Books would be garbage-collected when the Bookstore is removed. With a finalizer-only approach, we explicitly list and delete them. For a normal number of Books thats negligible and keeps the design consistent (one cleanup path).

**Reconcile trigger (watch) vs updating original in copy’s reconciliation.** We could either have the copies reconcile loop update the original `referenceCount`, or add a watch so that when a Book with `spec.copyOf` changes, we trigger a reconcile on the _original_ book. I went with the watcher so the originals reconcile is the single place that updates `referenceCount` to keep things cleaner and consistent.

**Edge case:** If the original Book is deleted and a copy still has `spec.copyOf` pointing at it, the copy is left with a dangling reference. Not handled specially today.

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

   The Makefile passes `WEBHOOK_CERT_PATH` into `--webhook-cert-path`. the manager expects `tls.crt` and `tls.key` in that directory. Leave it running in a terminal.

3. **Apply samples** (see below for the order).

### Samples I use for testing

Apply in this order (jerusalems book has a `copyOf` pointing at tel-aviv, so tel-aviv has to exist first):

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

## Missing things

- Unit/Integration tests are not finished ;)

## Open questions

- **Certs path:** Im not sure why I've needed to add the usage of ENV inside the make file, hopefully there is another way
- **Reference validation:** The `copyOf` reference validation works in tests but not in practice (e.g. when applying with kubectl). Haven't figured out why yet.
