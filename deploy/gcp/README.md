# Deploy Determined in Google Cloud Platform (GCP)

## Requirements
Download Service Account credentials and set the environment variable `GOOGLE_CREDENTIALS` to the path to the key json file.
`export GOOGLE_CREDENTIALS="/path/to/key.json"`

The Service Account to be used with Terraform should have the following permissions:
* Cloud SQL Admin
* Compute Admin
* Compute Network Admin
* Security Admin
* Service Account Admin
* Service Account User
* Service Networking Admin
* Storage Admin

The following GCP APIs must be enabled:
* [Cloud SQL Admin API](https://cloud.google.com/sql/docs/mysql/admin-api)
* [Service Networking API](https://cloud.google.com/service-infrastructure/docs/service-networking/getting-started)

If Terraform is creating the subnetwork, the VPC network being used must have an [IP range allocated](https://cloud.google.com/vpc/docs/configure-private-services-access#procedure) and a [private service connection created](https://cloud.google.com/vpc/docs/configure-private-services-access#creating-connection).

## Setup
1. Install [Terraform](https://learn.hashicorp.com/terraform/getting-started/install.html "Terraform Installation Instructions")
2. Clone the repository
3. In this directory, initialize Terraform
`terraform init`

*Note*
Terraform will save a `.tfstate` file in the directory it is run (i.e. the current directory), which is used to manage the state of the deployment. Since this directory is a git repository, there is a chance that this `.tfstate` file will be removed or modified depending on what the user is doing in the repo. For personal deployments, our recommendation is to copy this directory outside the repository and run Terraform there so the `.tfstate` file remains unchanged and can continue to be used for future updates. For production deployments, we recommend using a backend such as [GCS](https://www.terraform.io/docs/backends/types/gcs.html) with versions and state locking support.


## Update the `defaults.auto.tfvars` file
The `defaults.auto.tfvars` file contains configuration variables for the cluster. All required variables should be completed by the user (see table below for a description of the variables). All optional variables can be updated by the user as needed. When Terraform runs, it will apply the variables set in this file to the build.


## Deploy a Determined cluster in GCP
`terraform apply`

*Note*
Since Terraform manages state in a `.tfstate` file, if you re-run `terraform apply` in the same directory with updated variables, Terraform will simply update the existing resources instead of creating a new set of resources.


## Required Variables
| Argument                 | Description                                           | Default Value     |
|--------------------------|-------------------------------------------------------|-------------------|
| `project_id`             | Project id for the project.                           |                   |
| `region`                 | The region to create the resources in.                |                   |
| `identifier`             | An identifier string that will be appended to names   |                   |
| `det_version`            | The version or commit hash of Determined to deploy    |                   |
| `environment_image`      | The base image to use for deployment                  |                   |


## Optional Variables
Most standard deployments can leave the following variables as is.

| Argument                 | Description                                           | Default Value     |
|--------------------------|-------------------------------------------------------|-------------------|
| `creds`                  | Path to credentials json if not included in env.      | null              |
| `network`                | The name of the VPC network to create.                | to be created     |
| `subnetwork`             | The name of the subnetwork.                           | to be created     |
| `gcs_bucket`             | The GCS Bucket to use.                                | to be created     |
| `service_account_email`  | The service account to use.                           | to be created     |
| `zone`                   | The zone to create resources in.                      | `region`-a        |
| `create_static_ip`       | Whether to create a static external ip for the master.| true              |
| `create_database`        | Whether to create a separate database for the master. | true              |
| `master_machine_type`    | The instance type for the master instance.            | n1-standard-16    |
| `agent_machine_type`     | The instance type for the agent instances.            | n1-standard-32    |
| `gpu_type`               | The type of GPUs on the agent instances.              | nvidia-tesla-v100 |
| `gpu_num`                | The number of GPUs per agent instance.                | 8                 |
| `max_instances`          | The maximum number of agent instances at any time.    | 8                 |
| `agent_docker_network`   | The docker network to use for agent instances.        | host              |
| `master_docker_network`  | The docker network to use for the master instance.    | determined        |
| `max_idle_agent_period`  | The time an agent can stay idle before it is removed. | 5m                |
| `scheme`                 | The URI scheme.                                       | http              |
| `port`                   | The port on the master to communicate on.             | 8080              |

*Note*

The `service_account_email` referenced above is applied to each GCE instance to give them access to various resources. This Service Account is different than the Service Account used to deploy Terraform scripts. Ensure this service account has the following permissions:
* Compute Admin
* Service Account User

If setting the `gcs_bucket` in addition to the `service_account_email`, ensure that the service account has read/write access to the `gcs bucket`.


## De-provisioning the cluster
To bring down the cluster:
`terraform destroy`
