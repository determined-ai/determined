// This is a generated file.  Editing it will make you sad.

package schemas

import (
	"encoding/json"
)

var (
	textBindMountV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/bind-mount.json",
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
`)
	textCheckDataLayerCacheV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/check-data-layer-cache.json",
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
`)
	textCheckEpochNotUsedV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/check-epoch-not-used.json",
    "title": "CheckEpochNotUsed",
    "additionalProperties": {
        "$ref": "http://determined.ai/schemas/expconf/v1/check-epoch-not-used.json"
    },
    "items": {
        "$ref": "http://determined.ai/schemas/expconf/v1/check-epoch-not-used.json"
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
`)
	textCheckGlobalBatchSizeV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/check-global-batch-size.json",
    "title": "CheckGlobalBatchSize",
    "union": {
        "defaultMessage": "is neither a positive integer nor an int hyperparameter",
        "items": [
            {
                "unionKey": "const:type=int",
                "allOf": [
                    {
                        "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter-int.json"
                    },
                    {
                        "properties": {
                            "minval": {
                                "type": "number",
                                "minimum": 1
                            }
                        }
                    }
                ]
            },
            {
                "unionKey": "const:type=const",
                "allOf": [
                    {
                        "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter-const.json"
                    },
                    {
                        "properties": {
                            "val": {
                                "type": "number",
                                "minimum": 1
                            }
                        }
                    }
                ]
            },
            {
                "unionKey": "const:type=categorical",
                "allOf": [
                    {
                        "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter-categorical.json"
                    },
                    {
                        "properties": {
                            "vals": {
                                "type": "array",
                                "items": {
                                    "type": "integer",
                                    "minimum": 1
                                }
                            }
                        }
                    }
                ]
            },
            {
                "unionKey": "never",
                "type": "integer",
                "minimum": 1
            }
        ]
    }
}
`)
	textCheckGridHyperparameterV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/check-grid-hyperparameter.json",
    "title": "CheckGridHyperparameter",
    "union": {
        "items": [
            {
                "unionKey": "type:array",
                "type": "array",
                "items": {
                    "$ref": "http://determined.ai/schemas/expconf/v1/check-grid-hyperparameter.json"
                }
            },
            {
                "unionKey": "not:hasattr:type",
                "type": "object",
                "properties": {
                    "type": false
                },
                "additionalProperties": {
                    "$ref": "http://determined.ai/schemas/expconf/v1/check-grid-hyperparameter.json"
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
`)
	textCheckPositiveLengthV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/check-positive-length.json",
    "title": "CheckPositiveLength",
    "allOf": [
        {
            "$ref": "http://determined.ai/schemas/expconf/v1/length.json"
        },
        {
            "additionalProperties": {
                "type": "integer",
                "minimum": 1
            }
        }
    ]
}
`)
	textCheckpointStorageConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/checkpoint-storage.json",
    "title": "CheckpointStorageConfig",
    "union": {
        "defaultMessage": "is not an object where object[\"type\"] is one of 'shared_fs', 'hdfs', 's3', or 'gcs'",
        "items": [
            {
                "unionKey": "const:type=shared_fs",
                "$ref": "http://determined.ai/schemas/expconf/v1/shared-fs.json"
            },
            {
                "unionKey": "const:type=hdfs",
                "$ref": "http://determined.ai/schemas/expconf/v1/hdfs.json"
            },
            {
                "unionKey": "const:type=s3",
                "$ref": "http://determined.ai/schemas/expconf/v1/s3.json"
            },
            {
                "unionKey": "const:type=gcs",
                "$ref": "http://determined.ai/schemas/expconf/v1/gcs.json"
            }
        ]
    }
}
`)
	textDataLayerGCSConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/data-layer-gcs.json",
    "title": "DataLayerGCSConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "bucket",
        "bucket_directory_path"
    ],
    "properties": {
        "type": {
            "const": "gcs"
        },
        "bucket": {
            "type": "string"
        },
        "bucket_directory_path": {
            "type": "string"
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
            "$ref": "http://determined.ai/schemas/expconf/v1/check-data-layer-cache.json"
        }
    ]
}
`)
	textDataLayerS3ConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/data-layer-s3.json",
    "title": "DataLayerS3Config",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "bucket",
        "bucket_directory_path"
    ],
    "properties": {
        "type": {
            "const": "s3"
        },
        "bucket": {
            "type": "string"
        },
        "bucket_directory_path": {
            "type": "string"
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
            "$ref": "http://determined.ai/schemas/expconf/v1/check-data-layer-cache.json"
        }
    ]
}
`)
	textDataLayerSharedFSConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/data-layer-shared-fs.json",
    "title": "DataLayerSharedFSConfig",
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
`)
	textDataLayerConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/data-layer.json",
    "title": "DataLayerConfig",
    "union": {
        "defaultMessage": "is not an object where object[\"type\"] is one of 'shared_fs', 's3', or 'gcs'",
        "items": [
            {
                "unionKey": "const:type=shared_fs",
                "$ref": "http://determined.ai/schemas/expconf/v1/data-layer-shared-fs.json"
            },
            {
                "unionKey": "const:type=gcs",
                "$ref": "http://determined.ai/schemas/expconf/v1/data-layer-gcs.json"
            },
            {
                "unionKey": "const:type=s3",
                "$ref": "http://determined.ai/schemas/expconf/v1/data-layer-s3.json"
            }
        ]
    }
}
`)
	textEnvironmentImageV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/environment-image.json",
    "title": "EnvironmentImage",
    "union": {
        "defaultMessage": "is neither a string nor a map of cpu/gpu to strings",
        "items": [
            {
                "unionKey": "never",
                "type": "object",
                "additionalProperties": false,
                "required": [
                    "cpu",
                    "gpu"
                ],
                "properties": {
                    "cpu": {
                        "type": "string"
                    },
                    "gpu": {
                        "type": "string"
                    }
                }
            },
            {
                "unionKey": "never",
                "type": "string"
            }
        ]
    }
}
`)
	textEnvironmentVariablesV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/environment-variables.json",
    "title": "EnvironmentVariables",
    "union": {
        "defaultMessage": "is neither a list of strings nor a map of cpu/gpu to lists of strings",
        "items": [
            {
                "unionKey": "never",
                "type": "object",
                "additionalProperties": false,
                "properties": {
                    "cpu": {
                        "type": [
                            "array",
                            "null"
                        ],
                        "items": {
                            "type": "string"
                        },
                        "default": []
                    },
                    "gpu": {
                        "type": [
                            "array",
                            "null"
                        ],
                        "items": {
                            "type": "string"
                        },
                        "default": []
                    }
                }
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
`)
	textEnvironmentConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/environment.json",
    "title": "EnvironmentConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "properties": {
        "image": {
            "type": [
                "object",
                "null"
            ],
            "$ref": "http://determined.ai/schemas/expconf/v1/environment-image.json",
            "default": null
        },
        "environment_variables": {
            "type": [
                "object",
                "null"
            ],
            "$ref": "http://determined.ai/schemas/expconf/v1/environment-variables.json",
            "default": []
        },
        "ports": {
            "type": [
                "object",
                "null"
            ],
            "additionalProperties": {
                "type": "integer"
            },
            "default": {}
        },
        "force_pull_image": {
            "type": [
                "boolean",
                "null"
            ],
            "default": false
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
                                    "env_from": "container EnvFrom is not configurable",
                                    "env": "container Env is not configurable, set it in the experiment config",
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
        }
    }
}
`)
	textExperimentConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/experiment.json",
    "title": "ExperimentConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "hyperparameters",
        "searcher"
    ],
    "eventuallyRequired": [
        "checkpoint_storage"
    ],
    "properties": {
        "bind_mounts": {
            "type": [
                "array",
                "null"
            ],
            "items": {
                "$ref": "http://determined.ai/schemas/expconf/v1/bind-mount.json"
            },
            "default": []
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
            "$ref": "http://determined.ai/schemas/expconf/v1/checkpoint-storage.json",
            "default": "null"
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
            "$ref": "http://determined.ai/schemas/expconf/v1/data-layer.json",
            "default": {
                "type": "shared_fs"
            }
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
            "default": ""
        },
        "entrypoint": {
            "type": [
                "string",
                "null"
            ],
            "checks": {
                "entrypoint must be of the form \"module.submodule:ClassName\"": {
                    "pattern": "^[a-zA-Z0-9_.]+:[a-zA-Z0-9_]+$"
                }
            },
            "default": null
        },
        "environment": {
            "type": [
                "object",
                "null"
            ],
            "$ref": "http://determined.ai/schemas/expconf/v1/environment.json",
            "default": null
        },
        "hyperparameters": {
            "type": "object",
            "required": [
                "global_batch_size"
            ],
            "properties": {
                "global_batch_size": {
                    "$ref": "http://determined.ai/schemas/expconf/v1/check-global-batch-size.json"
                }
            },
            "additionalProperties": {
                "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter.json"
            }
        },
        "internal": {
            "type": [
                "object",
                "null"
            ],
            "default": null
        },
        "labels": {
            "type": [
                "array",
                "null"
            ],
            "items": {
                "type": "string"
            },
            "default": null
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
            "$ref": "http://determined.ai/schemas/expconf/v1/length.json",
            "default": {
                "batches": 0
            }
        },
        "min_validation_period": {
            "type": [
                "object",
                "null"
            ],
            "$ref": "http://determined.ai/schemas/expconf/v1/length.json",
            "default": {
                "batches": 0
            }
        },
        "optimizations": {
            "type": [
                "object",
                "null"
            ],
            "$ref": "http://determined.ai/schemas/expconf/v1/optimizations.json",
            "default": {}
        },
        "perform_initial_validation": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
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
            "additionalProperties": false,
            "properties": {
                "experiment_seed": {
                    "type": [
                        "integer",
                        "null"
                    ],
                    "default": null
                }
            },
            "default": {}
        },
        "resources": {
            "type": [
                "object",
                "null"
            ],
            "$ref": "http://determined.ai/schemas/expconf/v1/resources.json",
            "default": {}
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
            "$ref": "http://determined.ai/schemas/expconf/v1/searcher.json"
        },
        "security": {
            "type": "null",
            "default": "null"
        },
        "tensorboard_storage": {
            "type": "null",
            "default": "null"
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
                                "$ref": "http://determined.ai/schemas/expconf/v1/check-grid-hyperparameter.json"
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
                            "$ref": "http://determined.ai/schemas/expconf/v1/check-epoch-not-used.json"
                        },
                        "min_checkpoint_period": {
                            "$ref": "http://determined.ai/schemas/expconf/v1/check-epoch-not-used.json"
                        },
                        "searcher": {
                            "$ref": "http://determined.ai/schemas/expconf/v1/check-epoch-not-used.json"
                        }
                    }
                }
            }
        }
    ],
    "checks": {
        "must specify an entrypoint that references the trial class": {
            "conditional": {
                "$comment": "when internal.native is null, expect an entrypoint",
                "when": {
                    "properties": {
                        "internal": {
                            "properties": {
                                "native": {
                                    "type": "null"
                                }
                            }
                        }
                    }
                },
                "enforce": {
                    "not": {
                        "properties": {
                            "entrypoint": {
                                "type": "null"
                            }
                        }
                    }
                }
            }
        }
    }
}
`)
	textGCSConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/gcs.json",
    "title": "GCSConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "bucket"
    ],
    "properties": {
        "type": {
            "const": "gcs"
        },
        "bucket": {
            "type": "string"
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
`)
	textHDFSConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/hdfs.json",
    "title": "HDFSConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "hdfs_url",
        "hdfs_path"
    ],
    "properties": {
        "type": {
            "const": "hdfs"
        },
        "hdfs_url": {
            "type": "string"
        },
        "hdfs_path": {
            "type": "string",
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
`)
	textCategoricalHyperparameterV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/hyperparameter-categorical.json",
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
`)
	textConstHyperparameterV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/hyperparameter-const.json",
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
`)
	textDoubleHyperparameterV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/hyperparameter-double.json",
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
`)
	textIntHyperparameterV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/hyperparameter-int.json",
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
`)
	textLogHyperparameterV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/hyperparameter-log.json",
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
`)
	textHyperparameterV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/hyperparameter.json",
    "title": "Hyperparameter",
    "union": {
        "items": [
            {
                "unionKey": "const:type=int",
                "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter-int.json"
            },
            {
                "unionKey": "const:type=double",
                "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter-double.json"
            },
            {
                "unionKey": "const:type=log",
                "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter-log.json"
            },
            {
                "unionKey": "const:type=const",
                "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter-const.json"
            },
            {
                "unionKey": "const:type=categorical",
                "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter-categorical.json"
            },
            {
                "unionKey": "type:array",
                "type": "array",
                "items": {
                    "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter.json"
                }
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
                    "$ref": "http://determined.ai/schemas/expconf/v1/hyperparameter.json"
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
            }
        ]
    }
}
`)
	textInternalConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/internal.json",
    "title": "InternalConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "properties": {
        "native": {
            "type": [
                "array",
                "null"
            ],
            "items": {
                "type": "string"
            },
            "default": null
        }
    }
}
`)
	textLengthV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/length.json",
    "title": "Length",
    "union": {
        "defaultMessage": "a length object must have one attribute named \"batches\", \"records\", or \"epochs\"",
        "items": [
            {
                "unionKey": "hasattr:batches",
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
                "unionKey": "hasattr:records",
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
                "unionKey": "hasattr:epochs",
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
`)
	textOptimizationsConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/optimizations.json",
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
`)
	textResourcesConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/resources.json",
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
        "max_slots": {
            "type": [
                "integer",
                "null"
            ],
            "default": "null"
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
            "default": null
        },
        "shm_size": {
            "type": [
                "integer",
                "null"
            ],
            "default": "null"
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
`)
	textS3ConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/s3.json",
    "title": "S3Config",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "access_key",
        "bucket",
        "secret_key"
    ],
    "properties": {
        "type": {
            "const": "s3"
        },
        "access_key": {
            "type": "string"
        },
        "bucket": {
            "type": "string"
        },
        "secret_key": {
            "type": "string"
        },
        "endpoint_url": {
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
`)
	textAdaptiveASHASearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-adaptive-asha.json",
    "title": "AdaptiveASHASearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
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
            "type": "integer",
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
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "metric": {
            "type": "string"
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
`)
	textAdaptiveSimpleSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-adaptive-simple.json",
    "title": "AdaptiveSimpleSearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
        "max_trials",
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "adaptive_simple"
        },
        "max_trials": {
            "type": "integer",
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
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "metric": {
            "type": "string"
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
`)
	textAdaptiveSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-adaptive.json",
    "title": "AdaptiveSearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
        "budget",
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "adaptive"
        },
        "budget": {
            "$ref": "http://determined.ai/schemas/expconf/v1/length.json"
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
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "train_stragglers": {
            "type": [
                "boolean",
                "null"
            ],
            "default": true
        },
        "metric": {
            "type": "string"
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
    },
    "checks": {
        "max_length and budget must be specified in terms of the same unit": {
            "compareProperties": {
                "type": "same_units",
                "a": "max_length",
                "b": "budget"
            }
        },
        "budget must be greater than max_length": {
            "compareProperties": {
                "type": "length_a<length_b",
                "a": "max_length",
                "b": "budget"
            }
        }
    }
}
`)
	textAsyncHalvingSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-async-halving.json",
    "title": "AsyncHalvingSearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
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
            "type": "integer",
            "minimum": 1
        },
        "max_length": {
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "max_trials": {
            "type": "integer",
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
        "metric": {
            "type": "string"
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
`)
	textGridSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-grid.json",
    "title": "GridSearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "grid"
        },
        "max_length": {
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "metric": {
            "type": "string"
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
`)
	textPBTSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-pbt.json",
    "title": "PBTSearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
        "metric",
        "population_size",
        "length_per_round",
        "num_rounds",
        "replace_function",
        "explore_function"
    ],
    "properties": {
        "name": {
            "const": "pbt"
        },
        "population_size": {
            "type": "integer",
            "minimum": 1
        },
        "length_per_round": {
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "num_rounds": {
            "type": "integer",
            "minimum": 1
        },
        "replace_function": {
            "unionKey": "singleproperty",
            "union": {
                "items": [
                    {
                        "unionKey": "always",
                        "type": "object",
                        "additionalProperties": false,
                        "required": [
                            "truncate_fraction"
                        ],
                        "properties": {
                            "truncate_fraction": {
                                "type": "number",
                                "minimum": 0.0,
                                "maximum": 1.0
                            }
                        }
                    }
                ]
            }
        },
        "explore_function": {
            "type": "object",
            "additionalProperties": false,
            "required": [
                "resample_probability",
                "perturb_factor"
            ],
            "properties": {
                "resample_probability": {
                    "type": "number",
                    "minimum": 0.0,
                    "maximum": 1.0
                },
                "perturb_factor": {
                    "type": "number",
                    "minimum": 0.0,
                    "maximum": 1.0
                }
            }
        },
        "metric": {
            "type": "string"
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
`)
	textRandomSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-random.json",
    "title": "RandomSearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
        "max_trials",
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "random"
        },
        "max_trials": {
            "type": "integer",
            "minimum": 1
        },
        "max_length": {
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "metric": {
            "type": "string"
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
`)
	textSingleSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-single.json",
    "title": "SingleSearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
        "max_length",
        "metric"
    ],
    "properties": {
        "name": {
            "const": "single"
        },
        "max_length": {
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "metric": {
            "type": "string"
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
`)
	textSyncHalvingSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher-sync-halving.json",
    "title": "SyncHalvingSearcherConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "name",
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
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
        },
        "num_rungs": {
            "type": "integer",
            "minimum": 1
        },
        "max_length": {
            "$ref": "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
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
            "type": "string"
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
`)
	textSearcherConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/searcher.json",
    "title": "SearcherConfig",
    "union": {
        "defaultMessage": "is not an object where object[\"name\"] is one of 'single', 'random', 'grid', 'adaptive', 'adaptive_asha', 'adaptive_simple', or 'pbt'",
        "items": [
            {
                "unionKey": "const:name=single",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-single.json"
            },
            {
                "unionKey": "const:name=random",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-random.json"
            },
            {
                "unionKey": "const:name=grid",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-grid.json"
            },
            {
                "unionKey": "const:name=adaptive_asha",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-adaptive-asha.json"
            },
            {
                "unionKey": "const:name=adaptive_simple",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-adaptive-simple.json"
            },
            {
                "unionKey": "const:name=adaptive",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-adaptive.json"
            },
            {
                "unionKey": "const:name=pbt",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-pbt.json"
            },
            {
                "unionKey": "const:name=sync_halving",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-sync-halving.json"
            },
            {
                "unionKey": "const:name=async_halving",
                "$ref": "http://determined.ai/schemas/expconf/v1/searcher-async-halving.json"
            }
        ]
    }
}
`)
	textSharedFSConfigV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/shared-fs.json",
    "title": "SharedFSConfig",
    "type": "object",
    "additionalProperties": false,
    "required": [
        "type",
        "host_path"
    ],
    "properties": {
        "type": {
            "const": "shared_fs"
        },
        "host_path": {
            "type": "string"
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
`)
	textTestRootV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/test-root.json",
    "title": "TestRoot",
    "type": "object",
    "additionalProperties": false,
    "required": ["val_x"],
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
            "optionalRef": "http://determined.ai/schemas/expconf/v1/test-sub.json"
        },
        "sub_union": {
            "type": [
                "object",
                "null"
            ],
            "default": null,
            "optionalRef": "http://determined.ai/schemas/expconf/v1/test-union.json"
        },
        "runtime_defaultable": {
            "type": [
                "integer",
                "null"
            ],
            "default": null
        }
    }
}

`)
	textTestSubV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/test-sub.json",
    "title": "TestSub",
    "type": "object",
    "additionalProperties": false,
    "required": [],
    "properties": {
        "val_y": {
            "type": ["string", "null"],
            "default": "default_y"
        }
    }
}

`)
	textTestUnionAV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/test-union-a.json",
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
            "type": ["string", "null"],
            "default": "default-common-val"
        }
    }
}
`)
	textTestUnionBV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/test-union-b.json",
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
            "type": ["string", "null"],
            "default": "default-common-val"
        }
    }
}
`)
	textTestUnionV1 = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://determined.ai/schemas/expconf/v1/test-union.json",
    "title": "TestUnion",
    "union": {
        "defaultMessage": "bad test union",
        "items": [
            {
                "unionKey": "const:type=a",
                "$ref": "http://determined.ai/schemas/expconf/v1/test-union-a.json"
            },
            {
                "unionKey": "const:type=b",
                "$ref": "http://determined.ai/schemas/expconf/v1/test-union-b.json"
            }
        ]
    }
}
`)
	schemaBindMountV1                    interface{}
	schemaCheckDataLayerCacheV1          interface{}
	schemaCheckEpochNotUsedV1            interface{}
	schemaCheckGlobalBatchSizeV1         interface{}
	schemaCheckGridHyperparameterV1      interface{}
	schemaCheckPositiveLengthV1          interface{}
	schemaCheckpointStorageConfigV1      interface{}
	schemaDataLayerGCSConfigV1           interface{}
	schemaDataLayerS3ConfigV1            interface{}
	schemaDataLayerSharedFSConfigV1      interface{}
	schemaDataLayerConfigV1              interface{}
	schemaEnvironmentImageV1             interface{}
	schemaEnvironmentVariablesV1         interface{}
	schemaEnvironmentConfigV1            interface{}
	schemaExperimentConfigV1             interface{}
	schemaGCSConfigV1                    interface{}
	schemaHDFSConfigV1                   interface{}
	schemaCategoricalHyperparameterV1    interface{}
	schemaConstHyperparameterV1          interface{}
	schemaDoubleHyperparameterV1         interface{}
	schemaIntHyperparameterV1            interface{}
	schemaLogHyperparameterV1            interface{}
	schemaHyperparameterV1               interface{}
	schemaInternalConfigV1               interface{}
	schemaLengthV1                       interface{}
	schemaOptimizationsConfigV1          interface{}
	schemaResourcesConfigV1              interface{}
	schemaS3ConfigV1                     interface{}
	schemaAdaptiveASHASearcherConfigV1   interface{}
	schemaAdaptiveSimpleSearcherConfigV1 interface{}
	schemaAdaptiveSearcherConfigV1       interface{}
	schemaAsyncHalvingSearcherConfigV1   interface{}
	schemaGridSearcherConfigV1           interface{}
	schemaPBTSearcherConfigV1            interface{}
	schemaRandomSearcherConfigV1         interface{}
	schemaSingleSearcherConfigV1         interface{}
	schemaSyncHalvingSearcherConfigV1    interface{}
	schemaSearcherConfigV1               interface{}
	schemaSharedFSConfigV1               interface{}
	schemaTestRootV1                     interface{}
	schemaTestSubV1                      interface{}
	schemaTestUnionAV1                   interface{}
	schemaTestUnionBV1                   interface{}
	schemaTestUnionV1                    interface{}
	cachedSchemaMap                      map[string]interface{}
	cachedSchemaBytesMap                 map[string][]byte
)

func ParsedBindMountV1() interface{} {
	if schemaBindMountV1 != nil {
		return schemaBindMountV1
	}
	err := json.Unmarshal(textBindMountV1, &schemaBindMountV1)
	if err != nil {
		panic("invalid embedded json for BindMountV1")
	}
	return schemaBindMountV1
}

func ParsedCheckDataLayerCacheV1() interface{} {
	if schemaCheckDataLayerCacheV1 != nil {
		return schemaCheckDataLayerCacheV1
	}
	err := json.Unmarshal(textCheckDataLayerCacheV1, &schemaCheckDataLayerCacheV1)
	if err != nil {
		panic("invalid embedded json for CheckDataLayerCacheV1")
	}
	return schemaCheckDataLayerCacheV1
}

func ParsedCheckEpochNotUsedV1() interface{} {
	if schemaCheckEpochNotUsedV1 != nil {
		return schemaCheckEpochNotUsedV1
	}
	err := json.Unmarshal(textCheckEpochNotUsedV1, &schemaCheckEpochNotUsedV1)
	if err != nil {
		panic("invalid embedded json for CheckEpochNotUsedV1")
	}
	return schemaCheckEpochNotUsedV1
}

func ParsedCheckGlobalBatchSizeV1() interface{} {
	if schemaCheckGlobalBatchSizeV1 != nil {
		return schemaCheckGlobalBatchSizeV1
	}
	err := json.Unmarshal(textCheckGlobalBatchSizeV1, &schemaCheckGlobalBatchSizeV1)
	if err != nil {
		panic("invalid embedded json for CheckGlobalBatchSizeV1")
	}
	return schemaCheckGlobalBatchSizeV1
}

func ParsedCheckGridHyperparameterV1() interface{} {
	if schemaCheckGridHyperparameterV1 != nil {
		return schemaCheckGridHyperparameterV1
	}
	err := json.Unmarshal(textCheckGridHyperparameterV1, &schemaCheckGridHyperparameterV1)
	if err != nil {
		panic("invalid embedded json for CheckGridHyperparameterV1")
	}
	return schemaCheckGridHyperparameterV1
}

func ParsedCheckPositiveLengthV1() interface{} {
	if schemaCheckPositiveLengthV1 != nil {
		return schemaCheckPositiveLengthV1
	}
	err := json.Unmarshal(textCheckPositiveLengthV1, &schemaCheckPositiveLengthV1)
	if err != nil {
		panic("invalid embedded json for CheckPositiveLengthV1")
	}
	return schemaCheckPositiveLengthV1
}

func ParsedCheckpointStorageConfigV1() interface{} {
	if schemaCheckpointStorageConfigV1 != nil {
		return schemaCheckpointStorageConfigV1
	}
	err := json.Unmarshal(textCheckpointStorageConfigV1, &schemaCheckpointStorageConfigV1)
	if err != nil {
		panic("invalid embedded json for CheckpointStorageConfigV1")
	}
	return schemaCheckpointStorageConfigV1
}

func ParsedDataLayerGCSConfigV1() interface{} {
	if schemaDataLayerGCSConfigV1 != nil {
		return schemaDataLayerGCSConfigV1
	}
	err := json.Unmarshal(textDataLayerGCSConfigV1, &schemaDataLayerGCSConfigV1)
	if err != nil {
		panic("invalid embedded json for DataLayerGCSConfigV1")
	}
	return schemaDataLayerGCSConfigV1
}

func ParsedDataLayerS3ConfigV1() interface{} {
	if schemaDataLayerS3ConfigV1 != nil {
		return schemaDataLayerS3ConfigV1
	}
	err := json.Unmarshal(textDataLayerS3ConfigV1, &schemaDataLayerS3ConfigV1)
	if err != nil {
		panic("invalid embedded json for DataLayerS3ConfigV1")
	}
	return schemaDataLayerS3ConfigV1
}

func ParsedDataLayerSharedFSConfigV1() interface{} {
	if schemaDataLayerSharedFSConfigV1 != nil {
		return schemaDataLayerSharedFSConfigV1
	}
	err := json.Unmarshal(textDataLayerSharedFSConfigV1, &schemaDataLayerSharedFSConfigV1)
	if err != nil {
		panic("invalid embedded json for DataLayerSharedFSConfigV1")
	}
	return schemaDataLayerSharedFSConfigV1
}

func ParsedDataLayerConfigV1() interface{} {
	if schemaDataLayerConfigV1 != nil {
		return schemaDataLayerConfigV1
	}
	err := json.Unmarshal(textDataLayerConfigV1, &schemaDataLayerConfigV1)
	if err != nil {
		panic("invalid embedded json for DataLayerConfigV1")
	}
	return schemaDataLayerConfigV1
}

func ParsedEnvironmentImageV1() interface{} {
	if schemaEnvironmentImageV1 != nil {
		return schemaEnvironmentImageV1
	}
	err := json.Unmarshal(textEnvironmentImageV1, &schemaEnvironmentImageV1)
	if err != nil {
		panic("invalid embedded json for EnvironmentImageV1")
	}
	return schemaEnvironmentImageV1
}

func ParsedEnvironmentVariablesV1() interface{} {
	if schemaEnvironmentVariablesV1 != nil {
		return schemaEnvironmentVariablesV1
	}
	err := json.Unmarshal(textEnvironmentVariablesV1, &schemaEnvironmentVariablesV1)
	if err != nil {
		panic("invalid embedded json for EnvironmentVariablesV1")
	}
	return schemaEnvironmentVariablesV1
}

func ParsedEnvironmentConfigV1() interface{} {
	if schemaEnvironmentConfigV1 != nil {
		return schemaEnvironmentConfigV1
	}
	err := json.Unmarshal(textEnvironmentConfigV1, &schemaEnvironmentConfigV1)
	if err != nil {
		panic("invalid embedded json for EnvironmentConfigV1")
	}
	return schemaEnvironmentConfigV1
}

func ParsedExperimentConfigV1() interface{} {
	if schemaExperimentConfigV1 != nil {
		return schemaExperimentConfigV1
	}
	err := json.Unmarshal(textExperimentConfigV1, &schemaExperimentConfigV1)
	if err != nil {
		panic("invalid embedded json for ExperimentConfigV1")
	}
	return schemaExperimentConfigV1
}

func ParsedGCSConfigV1() interface{} {
	if schemaGCSConfigV1 != nil {
		return schemaGCSConfigV1
	}
	err := json.Unmarshal(textGCSConfigV1, &schemaGCSConfigV1)
	if err != nil {
		panic("invalid embedded json for GCSConfigV1")
	}
	return schemaGCSConfigV1
}

func ParsedHDFSConfigV1() interface{} {
	if schemaHDFSConfigV1 != nil {
		return schemaHDFSConfigV1
	}
	err := json.Unmarshal(textHDFSConfigV1, &schemaHDFSConfigV1)
	if err != nil {
		panic("invalid embedded json for HDFSConfigV1")
	}
	return schemaHDFSConfigV1
}

func ParsedCategoricalHyperparameterV1() interface{} {
	if schemaCategoricalHyperparameterV1 != nil {
		return schemaCategoricalHyperparameterV1
	}
	err := json.Unmarshal(textCategoricalHyperparameterV1, &schemaCategoricalHyperparameterV1)
	if err != nil {
		panic("invalid embedded json for CategoricalHyperparameterV1")
	}
	return schemaCategoricalHyperparameterV1
}

func ParsedConstHyperparameterV1() interface{} {
	if schemaConstHyperparameterV1 != nil {
		return schemaConstHyperparameterV1
	}
	err := json.Unmarshal(textConstHyperparameterV1, &schemaConstHyperparameterV1)
	if err != nil {
		panic("invalid embedded json for ConstHyperparameterV1")
	}
	return schemaConstHyperparameterV1
}

func ParsedDoubleHyperparameterV1() interface{} {
	if schemaDoubleHyperparameterV1 != nil {
		return schemaDoubleHyperparameterV1
	}
	err := json.Unmarshal(textDoubleHyperparameterV1, &schemaDoubleHyperparameterV1)
	if err != nil {
		panic("invalid embedded json for DoubleHyperparameterV1")
	}
	return schemaDoubleHyperparameterV1
}

func ParsedIntHyperparameterV1() interface{} {
	if schemaIntHyperparameterV1 != nil {
		return schemaIntHyperparameterV1
	}
	err := json.Unmarshal(textIntHyperparameterV1, &schemaIntHyperparameterV1)
	if err != nil {
		panic("invalid embedded json for IntHyperparameterV1")
	}
	return schemaIntHyperparameterV1
}

func ParsedLogHyperparameterV1() interface{} {
	if schemaLogHyperparameterV1 != nil {
		return schemaLogHyperparameterV1
	}
	err := json.Unmarshal(textLogHyperparameterV1, &schemaLogHyperparameterV1)
	if err != nil {
		panic("invalid embedded json for LogHyperparameterV1")
	}
	return schemaLogHyperparameterV1
}

func ParsedHyperparameterV1() interface{} {
	if schemaHyperparameterV1 != nil {
		return schemaHyperparameterV1
	}
	err := json.Unmarshal(textHyperparameterV1, &schemaHyperparameterV1)
	if err != nil {
		panic("invalid embedded json for HyperparameterV1")
	}
	return schemaHyperparameterV1
}

func ParsedInternalConfigV1() interface{} {
	if schemaInternalConfigV1 != nil {
		return schemaInternalConfigV1
	}
	err := json.Unmarshal(textInternalConfigV1, &schemaInternalConfigV1)
	if err != nil {
		panic("invalid embedded json for InternalConfigV1")
	}
	return schemaInternalConfigV1
}

func ParsedLengthV1() interface{} {
	if schemaLengthV1 != nil {
		return schemaLengthV1
	}
	err := json.Unmarshal(textLengthV1, &schemaLengthV1)
	if err != nil {
		panic("invalid embedded json for LengthV1")
	}
	return schemaLengthV1
}

func ParsedOptimizationsConfigV1() interface{} {
	if schemaOptimizationsConfigV1 != nil {
		return schemaOptimizationsConfigV1
	}
	err := json.Unmarshal(textOptimizationsConfigV1, &schemaOptimizationsConfigV1)
	if err != nil {
		panic("invalid embedded json for OptimizationsConfigV1")
	}
	return schemaOptimizationsConfigV1
}

func ParsedResourcesConfigV1() interface{} {
	if schemaResourcesConfigV1 != nil {
		return schemaResourcesConfigV1
	}
	err := json.Unmarshal(textResourcesConfigV1, &schemaResourcesConfigV1)
	if err != nil {
		panic("invalid embedded json for ResourcesConfigV1")
	}
	return schemaResourcesConfigV1
}

func ParsedS3ConfigV1() interface{} {
	if schemaS3ConfigV1 != nil {
		return schemaS3ConfigV1
	}
	err := json.Unmarshal(textS3ConfigV1, &schemaS3ConfigV1)
	if err != nil {
		panic("invalid embedded json for S3ConfigV1")
	}
	return schemaS3ConfigV1
}

func ParsedAdaptiveASHASearcherConfigV1() interface{} {
	if schemaAdaptiveASHASearcherConfigV1 != nil {
		return schemaAdaptiveASHASearcherConfigV1
	}
	err := json.Unmarshal(textAdaptiveASHASearcherConfigV1, &schemaAdaptiveASHASearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for AdaptiveASHASearcherConfigV1")
	}
	return schemaAdaptiveASHASearcherConfigV1
}

func ParsedAdaptiveSimpleSearcherConfigV1() interface{} {
	if schemaAdaptiveSimpleSearcherConfigV1 != nil {
		return schemaAdaptiveSimpleSearcherConfigV1
	}
	err := json.Unmarshal(textAdaptiveSimpleSearcherConfigV1, &schemaAdaptiveSimpleSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for AdaptiveSimpleSearcherConfigV1")
	}
	return schemaAdaptiveSimpleSearcherConfigV1
}

func ParsedAdaptiveSearcherConfigV1() interface{} {
	if schemaAdaptiveSearcherConfigV1 != nil {
		return schemaAdaptiveSearcherConfigV1
	}
	err := json.Unmarshal(textAdaptiveSearcherConfigV1, &schemaAdaptiveSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for AdaptiveSearcherConfigV1")
	}
	return schemaAdaptiveSearcherConfigV1
}

func ParsedAsyncHalvingSearcherConfigV1() interface{} {
	if schemaAsyncHalvingSearcherConfigV1 != nil {
		return schemaAsyncHalvingSearcherConfigV1
	}
	err := json.Unmarshal(textAsyncHalvingSearcherConfigV1, &schemaAsyncHalvingSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for AsyncHalvingSearcherConfigV1")
	}
	return schemaAsyncHalvingSearcherConfigV1
}

func ParsedGridSearcherConfigV1() interface{} {
	if schemaGridSearcherConfigV1 != nil {
		return schemaGridSearcherConfigV1
	}
	err := json.Unmarshal(textGridSearcherConfigV1, &schemaGridSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for GridSearcherConfigV1")
	}
	return schemaGridSearcherConfigV1
}

func ParsedPBTSearcherConfigV1() interface{} {
	if schemaPBTSearcherConfigV1 != nil {
		return schemaPBTSearcherConfigV1
	}
	err := json.Unmarshal(textPBTSearcherConfigV1, &schemaPBTSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for PBTSearcherConfigV1")
	}
	return schemaPBTSearcherConfigV1
}

func ParsedRandomSearcherConfigV1() interface{} {
	if schemaRandomSearcherConfigV1 != nil {
		return schemaRandomSearcherConfigV1
	}
	err := json.Unmarshal(textRandomSearcherConfigV1, &schemaRandomSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for RandomSearcherConfigV1")
	}
	return schemaRandomSearcherConfigV1
}

func ParsedSingleSearcherConfigV1() interface{} {
	if schemaSingleSearcherConfigV1 != nil {
		return schemaSingleSearcherConfigV1
	}
	err := json.Unmarshal(textSingleSearcherConfigV1, &schemaSingleSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for SingleSearcherConfigV1")
	}
	return schemaSingleSearcherConfigV1
}

func ParsedSyncHalvingSearcherConfigV1() interface{} {
	if schemaSyncHalvingSearcherConfigV1 != nil {
		return schemaSyncHalvingSearcherConfigV1
	}
	err := json.Unmarshal(textSyncHalvingSearcherConfigV1, &schemaSyncHalvingSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for SyncHalvingSearcherConfigV1")
	}
	return schemaSyncHalvingSearcherConfigV1
}

func ParsedSearcherConfigV1() interface{} {
	if schemaSearcherConfigV1 != nil {
		return schemaSearcherConfigV1
	}
	err := json.Unmarshal(textSearcherConfigV1, &schemaSearcherConfigV1)
	if err != nil {
		panic("invalid embedded json for SearcherConfigV1")
	}
	return schemaSearcherConfigV1
}

func ParsedSharedFSConfigV1() interface{} {
	if schemaSharedFSConfigV1 != nil {
		return schemaSharedFSConfigV1
	}
	err := json.Unmarshal(textSharedFSConfigV1, &schemaSharedFSConfigV1)
	if err != nil {
		panic("invalid embedded json for SharedFSConfigV1")
	}
	return schemaSharedFSConfigV1
}

func ParsedTestRootV1() interface{} {
	if schemaTestRootV1 != nil {
		return schemaTestRootV1
	}
	err := json.Unmarshal(textTestRootV1, &schemaTestRootV1)
	if err != nil {
		panic("invalid embedded json for TestRootV1")
	}
	return schemaTestRootV1
}

func ParsedTestSubV1() interface{} {
	if schemaTestSubV1 != nil {
		return schemaTestSubV1
	}
	err := json.Unmarshal(textTestSubV1, &schemaTestSubV1)
	if err != nil {
		panic("invalid embedded json for TestSubV1")
	}
	return schemaTestSubV1
}

func ParsedTestUnionAV1() interface{} {
	if schemaTestUnionAV1 != nil {
		return schemaTestUnionAV1
	}
	err := json.Unmarshal(textTestUnionAV1, &schemaTestUnionAV1)
	if err != nil {
		panic("invalid embedded json for TestUnionAV1")
	}
	return schemaTestUnionAV1
}

func ParsedTestUnionBV1() interface{} {
	if schemaTestUnionBV1 != nil {
		return schemaTestUnionBV1
	}
	err := json.Unmarshal(textTestUnionBV1, &schemaTestUnionBV1)
	if err != nil {
		panic("invalid embedded json for TestUnionBV1")
	}
	return schemaTestUnionBV1
}

func ParsedTestUnionV1() interface{} {
	if schemaTestUnionV1 != nil {
		return schemaTestUnionV1
	}
	err := json.Unmarshal(textTestUnionV1, &schemaTestUnionV1)
	if err != nil {
		panic("invalid embedded json for TestUnionV1")
	}
	return schemaTestUnionV1
}

func schemaBytesMap() map[string][]byte {
	if cachedSchemaBytesMap != nil {
		return cachedSchemaBytesMap
	}
	var url string
	cachedSchemaBytesMap = map[string][]byte{}
	url = "http://determined.ai/schemas/expconf/v1/bind-mount.json"
	cachedSchemaBytesMap[url] = textBindMountV1
	url = "http://determined.ai/schemas/expconf/v1/check-data-layer-cache.json"
	cachedSchemaBytesMap[url] = textCheckDataLayerCacheV1
	url = "http://determined.ai/schemas/expconf/v1/check-epoch-not-used.json"
	cachedSchemaBytesMap[url] = textCheckEpochNotUsedV1
	url = "http://determined.ai/schemas/expconf/v1/check-global-batch-size.json"
	cachedSchemaBytesMap[url] = textCheckGlobalBatchSizeV1
	url = "http://determined.ai/schemas/expconf/v1/check-grid-hyperparameter.json"
	cachedSchemaBytesMap[url] = textCheckGridHyperparameterV1
	url = "http://determined.ai/schemas/expconf/v1/check-positive-length.json"
	cachedSchemaBytesMap[url] = textCheckPositiveLengthV1
	url = "http://determined.ai/schemas/expconf/v1/checkpoint-storage.json"
	cachedSchemaBytesMap[url] = textCheckpointStorageConfigV1
	url = "http://determined.ai/schemas/expconf/v1/data-layer-gcs.json"
	cachedSchemaBytesMap[url] = textDataLayerGCSConfigV1
	url = "http://determined.ai/schemas/expconf/v1/data-layer-s3.json"
	cachedSchemaBytesMap[url] = textDataLayerS3ConfigV1
	url = "http://determined.ai/schemas/expconf/v1/data-layer-shared-fs.json"
	cachedSchemaBytesMap[url] = textDataLayerSharedFSConfigV1
	url = "http://determined.ai/schemas/expconf/v1/data-layer.json"
	cachedSchemaBytesMap[url] = textDataLayerConfigV1
	url = "http://determined.ai/schemas/expconf/v1/environment-image.json"
	cachedSchemaBytesMap[url] = textEnvironmentImageV1
	url = "http://determined.ai/schemas/expconf/v1/environment-variables.json"
	cachedSchemaBytesMap[url] = textEnvironmentVariablesV1
	url = "http://determined.ai/schemas/expconf/v1/environment.json"
	cachedSchemaBytesMap[url] = textEnvironmentConfigV1
	url = "http://determined.ai/schemas/expconf/v1/experiment.json"
	cachedSchemaBytesMap[url] = textExperimentConfigV1
	url = "http://determined.ai/schemas/expconf/v1/gcs.json"
	cachedSchemaBytesMap[url] = textGCSConfigV1
	url = "http://determined.ai/schemas/expconf/v1/hdfs.json"
	cachedSchemaBytesMap[url] = textHDFSConfigV1
	url = "http://determined.ai/schemas/expconf/v1/hyperparameter-categorical.json"
	cachedSchemaBytesMap[url] = textCategoricalHyperparameterV1
	url = "http://determined.ai/schemas/expconf/v1/hyperparameter-const.json"
	cachedSchemaBytesMap[url] = textConstHyperparameterV1
	url = "http://determined.ai/schemas/expconf/v1/hyperparameter-double.json"
	cachedSchemaBytesMap[url] = textDoubleHyperparameterV1
	url = "http://determined.ai/schemas/expconf/v1/hyperparameter-int.json"
	cachedSchemaBytesMap[url] = textIntHyperparameterV1
	url = "http://determined.ai/schemas/expconf/v1/hyperparameter-log.json"
	cachedSchemaBytesMap[url] = textLogHyperparameterV1
	url = "http://determined.ai/schemas/expconf/v1/hyperparameter.json"
	cachedSchemaBytesMap[url] = textHyperparameterV1
	url = "http://determined.ai/schemas/expconf/v1/internal.json"
	cachedSchemaBytesMap[url] = textInternalConfigV1
	url = "http://determined.ai/schemas/expconf/v1/length.json"
	cachedSchemaBytesMap[url] = textLengthV1
	url = "http://determined.ai/schemas/expconf/v1/optimizations.json"
	cachedSchemaBytesMap[url] = textOptimizationsConfigV1
	url = "http://determined.ai/schemas/expconf/v1/resources.json"
	cachedSchemaBytesMap[url] = textResourcesConfigV1
	url = "http://determined.ai/schemas/expconf/v1/s3.json"
	cachedSchemaBytesMap[url] = textS3ConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-adaptive-asha.json"
	cachedSchemaBytesMap[url] = textAdaptiveASHASearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-adaptive-simple.json"
	cachedSchemaBytesMap[url] = textAdaptiveSimpleSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-adaptive.json"
	cachedSchemaBytesMap[url] = textAdaptiveSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-async-halving.json"
	cachedSchemaBytesMap[url] = textAsyncHalvingSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-grid.json"
	cachedSchemaBytesMap[url] = textGridSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-pbt.json"
	cachedSchemaBytesMap[url] = textPBTSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-random.json"
	cachedSchemaBytesMap[url] = textRandomSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-single.json"
	cachedSchemaBytesMap[url] = textSingleSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher-sync-halving.json"
	cachedSchemaBytesMap[url] = textSyncHalvingSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/searcher.json"
	cachedSchemaBytesMap[url] = textSearcherConfigV1
	url = "http://determined.ai/schemas/expconf/v1/shared-fs.json"
	cachedSchemaBytesMap[url] = textSharedFSConfigV1
	url = "http://determined.ai/schemas/expconf/v1/test-root.json"
	cachedSchemaBytesMap[url] = textTestRootV1
	url = "http://determined.ai/schemas/expconf/v1/test-sub.json"
	cachedSchemaBytesMap[url] = textTestSubV1
	url = "http://determined.ai/schemas/expconf/v1/test-union-a.json"
	cachedSchemaBytesMap[url] = textTestUnionAV1
	url = "http://determined.ai/schemas/expconf/v1/test-union-b.json"
	cachedSchemaBytesMap[url] = textTestUnionBV1
	url = "http://determined.ai/schemas/expconf/v1/test-union.json"
	cachedSchemaBytesMap[url] = textTestUnionV1
	return cachedSchemaBytesMap
}
