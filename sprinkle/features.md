# Sprinkle API Features

## Selling points

(mostly by Angela)

**High-level API:**

* Natural Keras experience
* Natural Estimator experience
* Natural Lightning experience
* Same API for training and distributed batch inference

**Low-level API:**

* Modular experience; add functionality incrementally as you need it
* Deliver user benefits on day 2, rather than let them hit day 90 and realize
  that they wish they had started with PyTorchTrial

**New Possibilities**

* Spot instance support on day 1 for arbitrary workloads
  * (requires more master-side work, but the API is there)
* Zero-downtime to training while master upgrades

## Feature grid

Platform Features:

* `Metr`: metrics visualizations
* `Ckpt`: checkpoint tracking
  * enables continue training feature
* `Gang`: gang scheduling
* `AdpPbt`: adaptive + pbt searchers
* `HpS`: all other hp searchers
* `EpcP`: epoch-granularity pausing
* `AnyP`: anytime pause
  * enables performant spot instance support

Training loop features:

* `MVP`: `min_validation_period`
* `MCP`: `min_checkpoint_period`
* `DTrn`: automatic distributed training

Training loop features:

| Feature              |Metr|Ckpt|Gang|HpS|AdpPbt|EpcP|AnyP|DTrn|MVP|MCP|
| ---                  |----|----|----|---|------|----|----|----|---|---|
| Sprinkle Keras       | x  | x  | x  | E | E    | x  |    | x  |   |   |
| Sprinkle Estimators  | x  | x  | x  | E | E    | x  |    | x  |   |   |
| Sprinkle PTL         | x  | x  | x  | E | E    | x  |    | x  |   |   |
| PyTorchTrial         | x  | x  | x  | x | x    | x  | x  | x  | x | x |
| \*KerasTrial         | x  | x  | x  | x | x    | x  | x  | x  | x | x |
| \*EstimatorTrial     | x  | x  | x  | x | x    | x  | x  | x  | x | x |
| *Low-level API*      |    |    |    |   |      |    |    |    |   |   |
| no api calls at all  |    |    | x  | x |      |    |    | ?  | ? | ? |
| Metrics + checkpoint | x  | x  | x  | x |      |    |    | ?  | ? | ? |
| + resume on startup  | x  | x  | x  | x |      |    |    | ?  | ? | ? |
| + searcher API       | x  | x  | x  | x | x    |    |    | ?  | ? | ? |
| + preemption API     | x  | x  | x  | x | x    | x  | x  | ?  | ? | ? |

\* = deprecated in favor of Sprinkle version
E = "epoch-based" searchers only
? = training loop details are up to the user

## Shortcomings:

* Still a mismatch in how difficult it is to set up cluster vs how easy it is to
  get benefit from the cluster
