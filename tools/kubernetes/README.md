### Using this tool for local development

**TLDR:** Just run `./shared-cluster.sh connect`.

#### Prerequisites
1. Someone on has already run `shared-cluster.up` (ask how to change the cluster if you aren't backend).
2. You have the `gcloud` CLI installed and authenticated.

#### Connecting
If the cluster is up, all you need to know is to run `./shared-cluster.sh connect` and your local devcluster
will be talking to a remote GKE cluster.

#### Setting the shared cluster up
1. Run `./shared-cluster.sh up`. If you aren't on backend, please `s/backend/yourteam/g` in the script before
    running it.

#### How it works.

We launch a shared, private GKE cluster and a bastion instance for folks to connect through. Locally, `connect` will
launch a SOCKS5 tunnel and route all your traffic to the cluster through that. On the bastion instance, you will get
a random port allocated and a reverse tunnel that in-cluster traffic uses to talk back to your local.

If we want notebooks to work, we'll need another layer of proxies and to use
[-ProxyCommand](https://goteleport.com/blog/ssh-proxyjump-ssh-proxycommand/).
