# This is a generated file.  Editing it will make you sad.

import json

schemas = {
    "http://determined.ai/schemas/expconf/v0/azure.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/azure.json",
    "title": "AzureConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type"
    ],
    "eventuallyRequired": [
        "container"
    ],
    "eventually": {
        "checks": {
            "Exactly one of connection_string or account_url must be set": {
                "oneOf": [
                    {
                        "eventuallyRequired": [
                            "connection_string"
                        ]
                    },
                    {
                        "eventuallyRequired": [
                            "account_url"
                        ]
                    }
                ]
            }
        }
    },
    "checks": {
        "credential and connection_string must not both be set": {
            "not": {
                "required": [
                    "connection_string",
                    "credential"
                ],
                "properties": {
                    "connection_string": {
                        "type": "string"
                    },
                    "credential": {
                        "type": "string"
                    }
                }
            }
        }
    },
    "properties": {
        "type": {
            "const": "azure"
        },
        "container": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "connection_string": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "account_url": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "credential": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "save_experiment_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 0,
            "minimum": 0
        },
        "save_trial_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        },
        "save_trial_latest": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/bind-mount.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/bind-mount.json",
    "title": "BindMount",
    "additionalProperties": false,
    "required": [
        "host_path",
        "container_path"
    ],
    "type": "object",
    "properties": {
        "host_path": {
            "type": "string",
            "checks": {
                "host_path must be an absolute path": {
                    "pattern": "^/"
                }
            }
        },
        "container_path": {
            "type": "string",
            "checks": {
                "container_path must not be \".\"": {
                    "not": {
                        "const": "."
                    }
                }
            }
        },
        "read_only": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "propagation": {
            "type": [
                "string",
                "null"
            ],
            "default": "rprivate"
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/bind-mounts.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/bind-mounts.json",
    "title": "BindMountsConfig",
    "type": "array",
    "items": {
        "$ref": "http://determined.ai/schemas/expconf/v0/bind-mount.json"
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/check-data-layer-cache.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/check-data-layer-cache.json",
    "title": "CheckDataLayerCache",
    "checks": {
        "local_cache_container_path must be specified if local_cache_host_path is set": {
            "not": {
                "required": [
                    "local_cache_host_path"
                ],
                "properties": {
                    "local_cache_container_path": {
                        "type": "null"
                    },
                    "local_cache_host_path": {
                        "type": "string"
                    }
                }
            }
        },
        "local_cache_host_path must be specified if local_cache_container_path is set": {
            "not": {
                "required": [
                    "local_cache_container_path"
                ],
                "properties": {
                    "local_cache_container_path": {
                        "type": "string"
                    },
                    "local_cache_host_path": {
                        "type": "null"
                    }
                }
            }
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/check-epoch-not-used.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/check-epoch-not-used.json",
    "title": "CheckEpochNotUsed",
    "additionalProperties": {
        "$ref": "http://determined.ai/schemas/expconf/v0/check-epoch-not-used.json"
    },
    "items": {
        "$ref": "http://determined.ai/schemas/expconf/v0/check-epoch-not-used.json"
    },
    "checks": {
        "must specify the top-level records_per_epoch when this field is in terms of epochs": {
            "properties": {
                "epochs": {
                    "not": {
                        "type": "number"
                    }
                }
            }
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/check-grid-hyperparameter.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/check-grid-hyperparameter.json",
    "title": "CheckGridHyperparameter",
    "union": {
        "items": [
            {
                "unionKey": "type:array",
                "type": "array",
                "items": {
                    "$ref": "http://determined.ai/schemas/expconf/v0/check-grid-hyperparameter.json"
                }
            },
            {
                "unionKey": "not:hasattr:type",
                "type": "object",
                "properties": {
                    "type": false
                },
                "additionalProperties": {
                    "$ref": "http://determined.ai/schemas/expconf/v0/check-grid-hyperparameter.json"
                }
            },
            {
                "unionKey": "never",
                "not": {
                    "type": [
                        "object",
                        "array"
                    ]
                }
            },
            {
                "unionKey": "hasattr:type",
                "type": "object",
                "required": [
                    "type"
                ],
                "properties": {
                    "type": {
                        "type": "string"
                    }
                },
                "checks": {
                    "grid search is in use but count was not provided": {
                        "conditional": {
                            "$comment": "unless type is not double/log/int, expect non-null count",
                            "unless": {
                                "not": {
                                    "properties": {
                                        "type": {
                                            "enum": [
                                                "double",
                                                "log",
                                                "int"
                                            ]
                                        }
                                    }
                                }
                            },
                            "enforce": {
                                "not": {
                                    "properties": {
                                        "count": {
                                            "type": "null"
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        ]
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/check-positive-length.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/check-positive-length.json",
    "title": "CheckPositiveLength",
    "allOf": [
        {
            "$ref": "http://determined.ai/schemas/expconf/v0/length.json"
        },
        {
            "additionalProperties": {
                "type": "integer",
                "minimum": 1
            }
        }
    ]
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/checkpoint-storage.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/checkpoint-storage.json",
    "title": "CheckpointStorageConfig",
    "$comment": "this is a union of all possible properties, with validation for the common properties",
    "conditional": {
        "when": {
            "required": [
                "type"
            ]
        },
        "enforce": {
            "union": {
                "defaultMessage": "is not an object where object[\"type\"] is one of 'shared_fs', 'hdfs', 's3', 'gcs' or 'azure'",
                "items": [
                    {
                        "unionKey": "const:type=shared_fs",
                        "$ref": "http://determined.ai/schemas/expconf/v0/shared-fs.json"
                    },
                    {
                        "unionKey": "const:type=hdfs",
                        "$ref": "http://determined.ai/schemas/expconf/v0/hdfs.json"
                    },
                    {
                        "unionKey": "const:type=s3",
                        "$ref": "http://determined.ai/schemas/expconf/v0/s3.json"
                    },
                    {
                        "unionKey": "const:type=gcs",
                        "$ref": "http://determined.ai/schemas/expconf/v0/gcs.json"
                    },
                    {
                        "unionKey": "const:type=azure",
                        "$ref": "http://determined.ai/schemas/expconf/v0/azure.json"
                    }
                ]
            }
        }
    },
    "additionalProperties": false,
    "eventuallyRequired": [
        "type"
    ],
    "properties": {
        "access_key": true,
        "account_url": true,
        "bucket": true,
        "checkpoint_path": true,
        "connection_string": true,
        "container": true,
        "container_path": true,
        "credential": true,
        "endpoint_url": true,
        "prefix": true,
        "hdfs_path": true,
        "hdfs_url": true,
        "host_path": true,
        "propagation": true,
        "secret_key": true,
        "storage_path": true,
        "tensorboard_path": true,
        "type": true,
        "user": true,
        "save_experiment_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 0,
            "minimum": 0
        },
        "save_trial_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        },
        "save_trial_latest": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/data-layer-gcs.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/data-layer-gcs.json",
    "title": "GCSDataLayerConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type"
    ],
    "eventuallyRequired": [
        "bucket",
        "bucket_directory_path"
    ],
    "properties": {
        "type": {
            "const": "gcs"
        },
        "bucket": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "bucket_directory_path": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "local_cache_host_path": {
            "type": [
                "string",
                "null"
            ],
            "checks": {
                "local_cache_host_path must be an absolute path": {
                    "pattern": "^/"
                }
            },
            "default": null
        },
        "local_cache_container_path": {
            "type": [
                "string",
                "null"
            ],
            "checks": {
                "local_cache_container_path must be an absolute path": {
                    "pattern": "^/"
                }
            },
            "default": null
        }
    },
    "allOf": [
        {
            "$ref": "http://determined.ai/schemas/expconf/v0/check-data-layer-cache.json"
        }
    ]
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/data-layer-s3.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/data-layer-s3.json",
    "title": "S3DataLayerConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type"
    ],
    "eventuallyRequired": [
        "bucket",
        "bucket_directory_path"
    ],
    "properties": {
        "type": {
            "const": "s3"
        },
        "bucket": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "bucket_directory_path": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "local_cache_host_path": {
            "type": [
                "string",
                "null"
            ],
            "checks": {
                "local_cache_host_path must be an absolute path": {
                    "pattern": "^/"
                }
            },
            "default": null
        },
        "local_cache_container_path": {
            "type": [
                "string",
                "null"
            ],
            "checks": {
                "local_cache_container_path must be an absolute path": {
                    "pattern": "^/"
                }
            },
            "default": null
        },
        "access_key": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "secret_key": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "endpoint_url": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    },
    "allOf": [
        {
            "$ref": "http://determined.ai/schemas/expconf/v0/check-data-layer-cache.json"
        }
    ]
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/data-layer-shared-fs.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/data-layer-shared-fs.json",
    "title": "SharedFSDataLayerConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type"
    ],
    "properties": {
        "type": {
            "const": "shared_fs"
        },
        "host_storage_path": {
            "type": [
                "string",
                "null"
            ],
            "checks": {
                "host_storage_path must be an absolute path": {
                    "pattern": "^/"
                }
            },
            "default": null
        },
        "container_storage_path": {
            "type": [
                "string",
                "null"
            ],
            "checks": {
                "container_storage_path must be an absolute path": {
                    "pattern": "^/"
                }
            },
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/data-layer.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/data-layer.json",
    "title": "DataLayerConfig",
    "union": {
        "defaultMessage": "is not an object where object[\"type\"] is one of 'shared_fs', 's3', or 'gcs'",
        "items": [
            {
                "unionKey": "const:type=shared_fs",
                "$ref": "http://determined.ai/schemas/expconf/v0/data-layer-shared-fs.json"
            },
            {
                "unionKey": "const:type=gcs",
                "$ref": "http://determined.ai/schemas/expconf/v0/data-layer-gcs.json"
            },
            {
                "unionKey": "const:type=s3",
                "$ref": "http://determined.ai/schemas/expconf/v0/data-layer-s3.json"
            }
        ]
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/device.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/device.json",
    "title": "Device",
    "additionalProperties": false,
    "required": [
        "host_path",
        "container_path"
    ],
    "type": "object",
    "properties": {
        "host_path": {
            "type": "string"
        },
        "container_path": {
            "type": "string"
        },
        "mode": {
            "type": [
                "string",
                "null"
            ],
            "default": "mrw"
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/devices.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/devices.json",
    "title": "DevicesConfig",
    "type": "array",
    "items": {
        "$ref": "http://determined.ai/schemas/expconf/v0/device.json"
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/environment-image-map.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/environment-image-map.json",
    "title": "EnvironmentImageMap",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "eventuallyRequired": [
        "cpu",
        "cuda",
        "rocm"
    ],
    "properties": {
        "cpu": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "cuda": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "rocm": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "gpu": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/environment-image.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/environment-image.json",
    "title": "EnvironmentImage",
    "union": {
        "defaultMessage": "is neither a string nor a map of cpu, cuda, or rocm to strings",
        "items": [
            {
                "unionKey": "never",
                "$ref": "http://determined.ai/schemas/expconf/v0/environment-image-map.json"
            },
            {
                "unionKey": "never",
                "type": "string"
            }
        ]
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/environment-variables-map.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/environment-variables-map.json",
    "title": "EnvironmentVariablesMap",
    "type": "object",
    "additionalProperties": false,
    "properties": {
        "cpu": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        },
        "cuda": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        },
        "rocm": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        },
        "gpu": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/environment-variables.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/environment-variables.json",
    "title": "EnvironmentVariables",
    "union": {
        "defaultMessage": "is neither a list of strings nor a map of cpu, cuda, or rocm to lists of strings",
        "items": [
            {
                "unionKey": "never",
                "$ref": "http://determined.ai/schemas/expconf/v0/environment-variables-map.json"
            },
            {
                "unionKey": "never",
                "type": "array",
                "items": {
                    "type": "string"
                }
            }
        ]
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/environment.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/environment.json",
    "title": "EnvironmentConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "eventuallyRequired": [
        "image"
    ],
    "properties": {
        "image": {
            "type": [
                "object",
                "string",
                "null"
            ],
            "default": {},
            "optionalRef": "http://determined.ai/schemas/expconf/v0/environment-image.json"
        },
        "environment_variables": {
            "type": [
                "object",
                "array",
                "null"
            ],
            "default": [],
            "optionalRef": "http://determined.ai/schemas/expconf/v0/environment-variables.json"
        },
        "ports": {
            "type": [
                "object",
                "null"
            ],
            "default": {},
            "additionalProperties": {
                "type": "integer"
            }
        },
        "force_pull_image": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "registry_auth": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/registry-auth.json"
        },
        "add_capabilities": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        },
        "drop_capabilities": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        },
        "pod_spec": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "disallowProperties": {
                "name": "pod Name is not a configurable option",
                "name_space": "pod NameSpace is not a configurable option"
            },
            "properties": {
                "spec": {
                    "type": [
                        "object",
                        "null"
                    ],
                    "default": null,
                    "properties": {
                        "containers": {
                            "type": [
                                "array",
                                "null"
                            ],
                            "default": null,
                            "items": {
                                "type": "object",
                                "disallowProperties": {
                                    "image": "container Image is not configurable, set it in the experiment config",
                                    "command": "container Command is not configurable",
                                    "args": "container Args are not configurable",
                                    "working_dir": "container WorkingDir is not configurable",
                                    "ports": "container Ports are not configurable",
                                    "liveness_probe": "container LivenessProbe is not configurable",
                                    "readiness_probe": "container ReadinessProbe is not configurable",
                                    "startup_probe": "container StartupProbe is not configurable",
                                    "lifecycle": "container Lifecycle is not configurable",
                                    "termination_message_path": "container TerminationMessagePath is not configurable",
                                    "termination_message_policy": "container TerminationMessagePolicy is not configurable",
                                    "image_pull_policy": "container ImagePullPolicy is not configurable, set it in the experiment config",
                                    "security_context": "container SecurityContext is not configurable, set it in the experiment config"
                                }
                            }
                        }
                    }
                }
            }
        },
        "slurm": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/experiment.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/experiment.json",
    "title": "ExperimentConfig",
    "type": "object",
    "additionalProperties": false,
    "eventuallyRequired": [
        "checkpoint_storage",
        "entrypoint",
        "name",
        "hyperparameters",
        "reproducibility",
        "searcher"
    ],
    "properties": {
        "bind_mounts": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "optionalRef": "http://determined.ai/schemas/expconf/v0/bind-mounts.json"
        },
        "checkpoint_policy": {
            "enum": [
                null,
                "best",
                "all",
                "none"
            ],
            "default": "best"
        },
        "checkpoint_storage": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/checkpoint-storage.json"
        },
        "data": {
            "type": [
                "object",
                "null"
            ],
            "default": {}
        },
        "data_layer": {
            "type": [
                "object",
                "null"
            ],
            "default": {
                "type": "shared_fs"
            },
            "optionalRef": "http://determined.ai/schemas/expconf/v0/data-layer.json"
        },
        "debug": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "description": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "entrypoint": {
            "type": [
                "string",
                "array",
                "null"
            ],
            "items": {
                "type": "string"
            },
            "default": null
        },
        "environment": {
            "type": [
                "object",
                "null"
            ],
            "default": {},
            "optionalRef": "http://determined.ai/schemas/expconf/v0/environment.json"
        },
        "hyperparameters": {
            "type": [
                "object",
                "null"
            ],
            "default": {},
            "optionalRef": "http://determined.ai/schemas/expconf/v0/hyperparameters.json"
        },
        "internal": {
            "$comment": "allow forking pre-0.15.6 non-Native-API experiments",
            "type": "null",
            "default": null
        },
        "labels": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        },
        "max_restarts": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 0,
            "default": 5
        },
        "min_checkpoint_period": {
            "type": [
                "object",
                "null"
            ],
            "default": {
                "batches": 0
            },
            "optionalRef": "http://determined.ai/schemas/expconf/v0/length.json"
        },
        "min_validation_period": {
            "type": [
                "object",
                "null"
            ],
            "default": {
                "batches": 0
            },
            "optionalRef": "http://determined.ai/schemas/expconf/v0/length.json"
        },
        "name": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "optimizations": {
            "type": [
                "object",
                "null"
            ],
            "default": {},
            "optionalRef": "http://determined.ai/schemas/expconf/v0/optimizations.json"
        },
        "perform_initial_validation": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "profiling": {
            "type": [
                "object",
                "null"
            ],
            "default": {},
            "optionalRef": "http://determined.ai/schemas/expconf/v0/profiling.json"
        },
        "records_per_epoch": {
            "type": [
                "integer",
                "null"
            ],
            "default": 0
        },
        "reproducibility": {
            "type": [
                "object",
                "null"
            ],
            "default": {},
            "optionalRef": "http://determined.ai/schemas/expconf/v0/reproducibility.json"
        },
        "resources": {
            "type": [
                "object",
                "null"
            ],
            "default": {},
            "optionalRef": "http://determined.ai/schemas/expconf/v0/resources.json"
        },
        "scheduling_unit": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 1,
            "default": 100
        },
        "searcher": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/searcher.json"
        },
        "security": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/security.json"
        },
        "tensorboard_storage": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/tensorboard-storage.json"
        }
    },
    "allOf": [
        {
            "conditional": {
                "$comment": "when grid search is in use, expect hp counts",
                "when": {
                    "properties": {
                        "searcher": {
                            "properties": {
                                "name": {
                                    "const": "grid"
                                }
                            }
                        }
                    }
                },
                "enforce": {
                    "properties": {
                        "hyperparameters": {
                            "additionalProperties": {
                                "$ref": "http://determined.ai/schemas/expconf/v0/check-grid-hyperparameter.json"
                            }
                        }
                    }
                }
            }
        },
        {
            "conditional": {
                "$comment": "when records per epoch not set, forbid epoch lengths",
                "when": {
                    "properties": {
                        "records_per_epoch": {
                            "maximum": 0
                        }
                    }
                },
                "enforce": {
                    "properties": {
                        "min_validation_period": {
                            "$ref": "http://determined.ai/schemas/expconf/v0/check-epoch-not-used.json"
                        },
                        "min_checkpoint_period": {
                            "$ref": "http://determined.ai/schemas/expconf/v0/check-epoch-not-used.json"
                        },
                        "searcher": {
                            "$ref": "http://determined.ai/schemas/expconf/v0/check-epoch-not-used.json"
                        }
                    }
                }
            }
        }
    ]
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/gcs.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/gcs.json",
    "title": "GCSConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type"
    ],
    "eventuallyRequired": [
        "bucket"
    ],
    "properties": {
        "type": {
            "const": "gcs"
        },
        "bucket": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "save_experiment_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 0,
            "minimum": 0
        },
        "save_trial_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        },
        "save_trial_latest": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/hdfs.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/hdfs.json",
    "title": "HDFSConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type"
    ],
    "eventuallyRequired": [
        "hdfs_url",
        "hdfs_path"
    ],
    "properties": {
        "type": {
            "const": "hdfs"
        },
        "hdfs_url": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "hdfs_path": {
            "type": [
                "string",
                "null"
            ],
            "default": null,
            "checks": {
                "hdfs_path must be an absolute path": {
                    "pattern": "^/"
                }
            }
        },
        "user": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "save_experiment_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 0,
            "minimum": 0
        },
        "save_trial_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        },
        "save_trial_latest": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/hyperparameter-categorical.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/hyperparameter-categorical.json",
    "title": "CategoricalHyperparameter",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "vals"
    ],
    "properties": {
        "type": {
            "const": "categorical"
        },
        "vals": {
            "type": "array",
            "minLength": 1
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/hyperparameter-const.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/hyperparameter-const.json",
    "title": "ConstHyperparameter",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "val"
    ],
    "properties": {
        "type": {
            "const": "const"
        },
        "val": true
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/hyperparameter-double.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/hyperparameter-double.json",
    "title": "DoubleHyperparameter",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "minval",
        "maxval"
    ],
    "properties": {
        "type": {
            "const": "double"
        },
        "minval": {
            "type": "number"
        },
        "maxval": {
            "type": "number"
        },
        "count": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1
        }
    },
    "compareProperties": {
        "type": "a<b",
        "a": "minval",
        "b": "maxval"
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/hyperparameter-int.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/hyperparameter-int.json",
    "title": "IntHyperparameter",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "minval",
        "maxval"
    ],
    "properties": {
        "type": {
            "const": "int"
        },
        "minval": {
            "type": "integer"
        },
        "maxval": {
            "type": "integer"
        },
        "count": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1
        }
    },
    "compareProperties": {
        "type": "a<b",
        "a": "minval",
        "b": "maxval"
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/hyperparameter-log.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/hyperparameter-log.json",
    "title": "LogHyperparameter",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "minval",
        "maxval",
        "base"
    ],
    "properties": {
        "type": {
            "const": "log"
        },
        "minval": {
            "type": "number"
        },
        "maxval": {
            "type": "number"
        },
        "base": {
            "type": "number",
            "exclusiveMinimum": 0
        },
        "count": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1
        }
    },
    "compareProperties": {
        "type": "a<b",
        "a": "minval",
        "b": "maxval"
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/hyperparameter.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/hyperparameter.json",
    "title": "Hyperparameter",
    "union": {
        "items": [
            {
                "unionKey": "const:type=int",
                "$ref": "http://determined.ai/schemas/expconf/v0/hyperparameter-int.json"
            },
            {
                "unionKey": "const:type=double",
                "$ref": "http://determined.ai/schemas/expconf/v0/hyperparameter-double.json"
            },
            {
                "unionKey": "const:type=log",
                "$ref": "http://determined.ai/schemas/expconf/v0/hyperparameter-log.json"
            },
            {
                "unionKey": "const:type=const",
                "$ref": "http://determined.ai/schemas/expconf/v0/hyperparameter-const.json"
            },
            {
                "unionKey": "const:type=categorical",
                "$ref": "http://determined.ai/schemas/expconf/v0/hyperparameter-categorical.json"
            },
            {
                "unionKey": "always",
                "type": "object",
                "checks": {
                    "if a hyperparameter object's [\"type\"] is set, it must be one of \"int\", \"double\", \"log\", const\", or \"categorical\"": {
                        "properties": {
                            "type": false
                        }
                    }
                },
                "additionalProperties": {
                    "$ref": "http://determined.ai/schemas/expconf/v0/hyperparameter.json"
                }
            },
            {
                "unionKey": "never",
                "not": {
                    "type": "object"
                }
            }
        ]
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/hyperparameters.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/hyperparameters.json",
    "title": "Hyperparameters",
    "type": "object",
    "additionalProperties": {
        "$ref": "http://determined.ai/schemas/expconf/v0/hyperparameter.json"
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/kerberos.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/kerberos.json",
    "title": "KerberosConfig",
    "$comment": "KerberosConfig has not been used in a very long time",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "config_file"
    ],
    "properties": {
        "config_file": {
            "type": "string"
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/length.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/length.json",
    "title": "Length",
    "union": {
        "defaultMessage": "a length object must have one attribute named \"batches\", \"records\", or \"epochs\"",
        "items": [
            {
                "unionKey": "singleproperty:batches",
                "type": "object",
                "additionalProperties": false,
                "required": [
                    "batches"
                ],
                "properties": {
                    "batches": {
                        "type": "integer",
                        "minimum": 0
                    }
                }
            },
            {
                "unionKey": "singleproperty:records",
                "type": "object",
                "additionalProperties": false,
                "required": [
                    "records"
                ],
                "properties": {
                    "records": {
                        "type": "integer",
                        "minimum": 0
                    }
                }
            },
            {
                "unionKey": "singleproperty:epochs",
                "type": "object",
                "additionalProperties": false,
                "required": [
                    "epochs"
                ],
                "properties": {
                    "epochs": {
                        "type": "integer",
                        "minimum": 0
                    }
                }
            }
        ]
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/optimizations.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/optimizations.json",
    "title": "OptimizationsConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "properties": {
        "aggregation_frequency": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 1,
            "default": 1
        },
        "auto_tune_tensor_fusion": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "average_aggregated_gradients": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "average_training_metrics": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "gradient_compression": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "grad_updates_size_file": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "mixed_precision": {
            "enum": [
                null,
                "O0",
                "O1",
                "O2",
                "O3"
            ],
            "default": "O0",
            "checks": {
                "mixed_precision should be a string starting with an uppercase letter 'O'": {
                    "pattern": "^O"
                }
            }
        },
        "tensor_fusion_cycle_time": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 0,
            "default": 5
        },
        "tensor_fusion_threshold": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 0,
            "default": 64
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/profiling.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/profiling.json",
    "title": "ProfilingConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "properties": {
        "enabled": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "begin_on_batch": {
            "type": [
                "integer",
                "null"
            ],
            "default": 0,
            "minimum": 0
        },
        "end_after_batch": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 0
        },
        "sync_timings": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        }
    },
    "compareProperties": {
        "type": "a<=b",
        "a": "begin_on_batch",
        "b": "end_after_batch"
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/registry-auth.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/registry-auth.json",
    "title": "RegistryAuth",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "properties": {
        "username": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "password": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "auth": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "email": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "serveraddress": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "identitytoken": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "registrytoken": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/reproducibility.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/reproducibility.json",
    "title": "ReproducibilityConfig",
    "type": "object",
    "additionalProperties": false,
    "eventuallyRequired": [
        "experiment_seed"
    ],
    "properties": {
        "experiment_seed": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 0
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/resources.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/resources.json",
    "title": "ResourcesConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "properties": {
        "agent_label": {
            "type": [
                "string",
                "null"
            ],
            "default": ""
        },
        "devices": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "optionalRef": "http://determined.ai/schemas/expconf/v0/devices.json"
        },
        "max_slots": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "native_parallel": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "priority": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 1,
            "maximum": 99,
            "default": null
        },
        "resource_pool": {
            "type": [
                "string",
                "null"
            ],
            "default": ""
        },
        "shm_size": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "slots": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "slots_per_trial": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 0,
            "default": 1
        },
        "weight": {
            "type": [
                "number",
                "null"
            ],
            "default": 1
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/s3.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/s3.json",
    "title": "S3Config",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type"
    ],
    "eventuallyRequired": [
        "bucket"
    ],
    "properties": {
        "type": {
            "const": "s3"
        },
        "access_key": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "bucket": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "secret_key": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "endpoint_url": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "prefix": {
            "type": [
                "string",
                "null"
            ],
            "checks": {
                "prefix cannot contain /../": {
                    "not": {
                        "anyOf": [
                            {
                                "type": "string",
                                "pattern": "/\\.\\./"
                            },
                            {
                                "type": "string",
                                "pattern": "^\\.\\./"
                            },
                            {
                                "type": "string",
                                "pattern": "/\\.\\.$"
                            },
                            {
                                "type": "string",
                                "pattern": "^\\.\\.$"
                            }
                        ]
                    }
                }
            },
            "default": null
        },
        "save_experiment_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 0,
            "minimum": 0
        },
        "save_trial_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        },
        "save_trial_latest": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json",
    "title": "AdaptiveASHAConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name"
    ],
    "eventuallyRequired": [
        "max_length",
        "max_trials",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "adaptive_asha"
        },
        "bracket_rungs": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "integer"
            }
        },
        "max_trials": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1
        },
        "mode": {
            "enum": [
                null,
                "aggressive",
                "standard",
                "conservative"
            ],
            "default": "standard"
        },
        "divisor": {
            "type": [
                "number",
                "null"
            ],
            "exclusiveMinimum": 1,
            "default": 4
        },
        "max_rungs": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 1,
            "default": 5
        },
        "max_concurrent_trials": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 0,
            "default": 0
        },
        "max_length": {
            "type": [
                "object",
                "integer",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/searcher-length.json"
        },
        "stop_once": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$comment": "this is an EOL searcher, not to be used in new experiments",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json",
    "title": "AdaptiveSimpleConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name"
    ],
    "eventuallyRequired": [
        "max_trials",
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "adaptive_simple"
        },
        "max_trials": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1,
            "maximum": 2000
        },
        "mode": {
            "enum": [
                null,
                "aggressive",
                "standard",
                "conservative"
            ],
            "default": "standard"
        },
        "divisor": {
            "type": [
                "number",
                "null"
            ],
            "exclusiveMinimum": 1,
            "default": 4
        },
        "max_rungs": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 1,
            "default": 5
        },
        "max_length": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/check-positive-length.json"
        },
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-adaptive.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$comment": "this is an EOL searcher, not to be used in new experiments",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-adaptive.json",
    "title": "AdaptiveConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name"
    ],
    "eventuallyRequired": [
        "budget",
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "adaptive"
        },
        "budget": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/length.json"
        },
        "bracket_rungs": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "integer"
            }
        },
        "mode": {
            "enum": [
                null,
                "aggressive",
                "standard",
                "conservative"
            ],
            "default": "standard"
        },
        "divisor": {
            "type": [
                "number",
                "null"
            ],
            "exclusiveMinimum": 1,
            "default": 4
        },
        "max_rungs": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 1,
            "default": 5
        },
        "max_length": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/check-positive-length.json"
        },
        "train_stragglers": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-async-halving.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-async-halving.json",
    "title": "AsyncHalvingConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name"
    ],
    "eventuallyRequired": [
        "num_rungs",
        "max_length",
        "max_trials",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "async_halving"
        },
        "num_rungs": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1
        },
        "max_length": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/check-positive-length.json"
        },
        "max_trials": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1
        },
        "divisor": {
            "type": [
                "number",
                "null"
            ],
            "exclusiveMinimum": 1,
            "default": 4
        },
        "max_concurrent_trials": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 0,
            "default": 0
        },
        "stop_once": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
        },
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-grid.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-grid.json",
    "title": "GridConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name"
    ],
    "eventuallyRequired": [
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "grid"
        },
        "max_concurrent_trials": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 0,
            "default": 0
        },
        "max_length": {
            "type": [
                "object",
                "integer",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/searcher-length.json"
        },
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-length.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-length.json",
    "title": "SearcherLength",
    "$comment": "SearcherLength is either a positive Length or a positive integer",
    "union": {
        "items": [
            {
                "unionKey": "not:type:object",
                "type": "integer",
                "minimum": 0
            },
            {
                "unionKey": "always",
                "$ref": "http://determined.ai/schemas/expconf/v0/check-positive-length.json"
            }
        ]
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-random.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-random.json",
    "title": "RandomConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name"
    ],
    "eventuallyRequired": [
        "max_trials",
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "random"
        },
        "max_concurrent_trials": {
            "type": [
                "integer",
                "null"
            ],
            "minimum": 0,
            "default": 0
        },
        "max_trials": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1
        },
        "max_length": {
            "type": [
                "object",
                "integer",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/searcher-length.json"
        },
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-single.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-single.json",
    "title": "SingleConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name"
    ],
    "eventuallyRequired": [
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "single"
        },
        "max_length": {
            "type": [
                "object",
                "integer",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/searcher-length.json"
        },
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$comment": "this is an EOL searcher, not to be used in new experiments",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json",
    "title": "SyncHalvingConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name"
    ],
    "eventuallyRequired": [
        "num_rungs",
        "max_length",
        "budget",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "sync_halving"
        },
        "budget": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/check-positive-length.json"
        },
        "num_rungs": {
            "type": [
                "integer",
                "null"
            ],
            "default": null,
            "minimum": 1
        },
        "max_length": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/check-positive-length.json"
        },
        "divisor": {
            "type": [
                "number",
                "null"
            ],
            "exclusiveMinimum": 1,
            "default": 4
        },
        "train_stragglers": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/searcher.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/searcher.json",
    "title": "SearcherConfig",
    "$comment": "this is a union of all possible properties, with validation for the common properties",
    "conditional": {
        "when": {
            "required": [
                "name"
            ]
        },
        "enforce": {
            "union": {
                "defaultMessage": "is not an object where object[\"name\"] is one of 'single', 'random', 'grid', 'adaptive_asha', or 'pbt'",
                "items": [
                    {
                        "unionKey": "const:name=single",
                        "$ref": "http://determined.ai/schemas/expconf/v0/searcher-single.json"
                    },
                    {
                        "unionKey": "const:name=random",
                        "$ref": "http://determined.ai/schemas/expconf/v0/searcher-random.json"
                    },
                    {
                        "unionKey": "const:name=grid",
                        "$ref": "http://determined.ai/schemas/expconf/v0/searcher-grid.json"
                    },
                    {
                        "unionKey": "const:name=adaptive_asha",
                        "$ref": "http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json"
                    },
                    {
                        "unionKey": "const:name=async_halving",
                        "$ref": "http://determined.ai/schemas/expconf/v0/searcher-async-halving.json"
                    },
                    {
                        "$comment": "this is an EOL searcher, not to be used in new experiments",
                        "unionKey": "const:name=adaptive",
                        "$ref": "http://determined.ai/schemas/expconf/v0/searcher-adaptive.json"
                    },
                    {
                        "$comment": "this is an EOL searcher, not to be used in new experiments",
                        "unionKey": "const:name=adaptive_simple",
                        "$ref": "http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json"
                    },
                    {
                        "$comment": "this is an EOL searcher, not to be used in new experiments",
                        "unionKey": "const:name=sync_halving",
                        "$ref": "http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json"
                    }
                ]
            }
        }
    },
    "additionalProperties": false,
    "eventuallyRequired": [
        "name",
        "metric"
    ],
    "properties": {
        "bracket_rungs": true,
        "divisor": true,
        "max_concurrent_trials": true,
        "max_length": true,
        "max_rungs": true,
        "max_trials": true,
        "mode": true,
        "name": true,
        "num_rungs": true,
        "stop_once": true,
        "metric": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "smaller_is_better": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "source_trial_id": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "source_checkpoint_uuid": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "budget": true,
        "train_stragglers": true
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/security.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/security.json",
    "title": "SecurityConfig",
    "$comment": "SecurityConfig has not been used in a very long time",
    "type": "object",
    "additionalProperties": false,
    "properties": {
        "kerberos": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/kerberos.json"
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/shared-fs.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/shared-fs.json",
    "title": "SharedFSConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type"
    ],
    "eventuallyRequired": [
        "host_path"
    ],
    "properties": {
        "type": {
            "const": "shared_fs"
        },
        "host_path": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "storage_path": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "propagation": {
            "type": [
                "string",
                "null"
            ],
            "default": "rprivate"
        },
        "container_path": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "checkpoint_path": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "tensorboard_path": {
            "type": [
                "string",
                "null"
            ],
            "default": null
        },
        "save_experiment_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 0,
            "minimum": 0
        },
        "save_trial_best": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        },
        "save_trial_latest": {
            "type": [
                "integer",
                "null"
            ],
            "default": 1,
            "minimum": 0
        }
    },
    "checks": {
        "storage_path must either be a relative directory or a subdirectory of host_path": {
            "compareProperties": {
                "type": "a_is_subdir_of_b",
                "a": "storage_path",
                "b": "host_path"
            }
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/tensorboard-storage.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/tensorboard-storage.json",
    "title": "TensorboardStorageConfig",
    "$comment": "TensorboardStorageConfig has not been used in a very long time",
    "union": {
        "defaultMessage": "this field is deprecated and will be ignored",
        "items": [
            {
                "unionKey": "never",
                "$ref": "http://determined.ai/schemas/expconf/v0/shared-fs.json"
            },
            {
                "unionKey": "never",
                "$ref": "http://determined.ai/schemas/expconf/v0/hdfs.json"
            },
            {
                "unionKey": "never",
                "$ref": "http://determined.ai/schemas/expconf/v0/s3.json"
            },
            {
                "unionKey": "never",
                "$ref": "http://determined.ai/schemas/expconf/v0/gcs.json"
            }
        ]
    },
    "disallowProperties": {
        "save_experiment_best": "this field is deprecated and will be ignored",
        "save_trial_best": "this field is deprecated and will be ignored",
        "save_trial_latest": "this field is deprecated and will be ignored"
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/test-root.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/test-root.json",
    "title": "TestRoot",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "val_x"
    ],
    "properties": {
        "val_x": {
            "type": "integer"
        },
        "sub_obj": {
            "type": [
                "object",
                "null"
            ],
            "default": {},
            "optionalRef": "http://determined.ai/schemas/expconf/v0/test-sub.json"
        },
        "sub_union": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v0/test-union.json"
        },
        "runtime_defaultable": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        },
        "defaulted_array": {
            "type": [
                "array",
                "null"
            ],
            "default": [],
            "items": {
                "type": "string"
            }
        },
        "nodefault_array": {
            "type": [
                "array",
                "null"
            ],
            "default": null,
            "items": {
                "type": "string"
            }
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/test-sub.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/test-sub.json",
    "title": "TestSub",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "properties": {
        "val_y": {
            "type": [
                "string",
                "null"
            ],
            "default": "default_y"
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/test-union-a.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/test-union-a.json",
    "title": "TestUnionA",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "val_a"
    ],
    "properties": {
        "type": {
            "const": "a"
        },
        "val_a": {
            "type": "integer"
        },
        "common_val": {
            "type": [
                "string",
                "null"
            ],
            "default": "default-common-val"
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/test-union-b.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/test-union-b.json",
    "title": "TestUnionB",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "val_b"
    ],
    "properties": {
        "type": {
            "const": "b"
        },
        "val_b": {
            "type": "integer"
        },
        "common_val": {
            "type": [
                "string",
                "null"
            ],
            "default": "default-common-val"
        }
    }
}

"""
    ),
    "http://determined.ai/schemas/expconf/v0/test-union.json": json.loads(
        r"""
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v0/test-union.json",
    "title": "TestUnion",
    "union": {
        "defaultMessage": "bad test union",
        "items": [
            {
                "unionKey": "const:type=a",
                "$ref": "http://determined.ai/schemas/expconf/v0/test-union-a.json"
            },
            {
                "unionKey": "const:type=b",
                "$ref": "http://determined.ai/schemas/expconf/v0/test-union-b.json"
            }
        ]
    }
}

"""
    ),
}
