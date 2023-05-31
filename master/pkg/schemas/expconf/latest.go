package expconf

// This file defines the latest version of each config, which should be used throughout the system.

type (
	AdaptiveASHAConfig        = AdaptiveASHAConfigV0
	AsyncHalvingConfig        = AsyncHalvingConfigV0
	AzureConfig               = AzureConfigV0
	BindMount                 = BindMountV0
	BindMountsConfig          = BindMountsConfigV0
	CategoricalHyperparameter = CategoricalHyperparameterV0
	CheckpointStorageConfig   = CheckpointStorageConfigV0
	ConstHyperparameter       = ConstHyperparameterV0
	CustomConfig              = CustomConfigV0
	DevicesConfig             = DevicesConfigV0
	Device                    = DeviceV0
	DoubleHyperparameter      = DoubleHyperparameterV0
	Entrypoint                = EntrypointV0
	EnvironmentConfig         = EnvironmentConfigV0
	EnvironmentImageMap       = EnvironmentImageMapV0
	EnvironmentVariablesMap   = EnvironmentVariablesMapV0
	ExperimentConfig          = ExperimentConfigV0
	GCSConfig                 = GCSConfigV0
	GridConfig                = GridConfigV0
	Hyperparameter            = HyperparameterV0
	Hyperparameters           = HyperparametersV0
	IntHyperparameter         = IntHyperparameterV0
	Labels                    = LabelsV0
	Length                    = LengthV0
	LogHyperparameter         = LogHyperparameterV0
	OptimizationsConfig       = OptimizationsConfigV0
	ProfilingConfig           = ProfilingConfigV0
	RandomConfig              = RandomConfigV0
	ReproducibilityConfig     = ReproducibilityConfigV0
	ResourcesConfig           = ResourcesConfigV0
	S3Config                  = S3ConfigV0
	SearcherConfig            = SearcherConfigV0
	SharedFSConfig            = SharedFSConfigV0
	SingleConfig              = SingleConfigV0
	SlurmConfig               = SlurmConfigV0
	PbsConfig                 = PbsConfigV0
	ProxyPort                 = ProxyPortV0
	ProxyPortsConfig          = ProxyPortsConfigV0
)

// These are EOL searchers, not to be used in new experiments.
type AdaptiveConfig = AdaptiveConfigV0

type (
	AdaptiveSimpleConfig = AdaptiveSimpleConfigV0
	SyncHalvingConfig    = SyncHalvingConfigV0
)
