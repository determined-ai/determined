import { act, render } from '@testing-library/react';
import { useModal } from 'hew/Modal';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { useLayoutEffect } from 'react';

import {
  decodeTrialResponseToTrialDetails,
  mapV1GetExperimentDetailsResponse,
} from 'services/decoder';

import ExperimentContinueModalComponent, {
  ContinueExperimentType,
  Props,
} from './ExperimentContinueModal';
import { ThemeProvider } from './ThemeProvider';

const mockUseFeature = vi.hoisted(() => vi.fn());
vi.mock('hooks/useFeature', () => {
  return {
    default: () => ({
      isOn: mockUseFeature,
    }),
  };
});

const mockTrial = decodeTrialResponseToTrialDetails({
  trial: {
    bestCheckpoint: {
      endTime: '2024-07-18T15:32:05.859962Z',
      metadata: null,
      resources: {
        'metadata.json': '26',
        'state': '9',
      },
      state: 'STATE_COMPLETED',
      totalBatches: 1,
      uuid: '71109b0b-7306-4be1-bfc1-4f5980695f7d',
    },
    bestValidation: {
      endTime: '2024-07-18T15:32:05.576988Z',
      metrics: {
        avgMetrics: {
          x: 2,
        },
        batchMetrics: [],
      },
      numInputs: 0,
      totalBatches: 1,
    },
    checkpointCount: 1,
    endTime: '2024-07-18T15:32:06.853844Z',
    experimentId: 7823,
    hparams: {
      increment_by: 2,
    },
    id: 54176,
    latestValidation: {
      endTime: '2024-07-18T15:32:05.576988Z',
      metrics: {
        avgMetrics: {
          x: 2,
        },
        batchMetrics: [],
      },
      numInputs: 0,
      totalBatches: 1,
    },
    restarts: 0,
    runnerState: '',
    searcherMetricValue: 0,
    startTime: '2024-07-18T15:31:01.793136Z',
    state: 'STATE_COMPLETED',
    summaryMetrics: {
      validation_metrics: {
        x: {
          count: 1,
          last: 2,
          max: 2,
          mean: 2,
          min: 2,
          sum: 2,
          type: 'number',
        },
      },
    },
    taskId: '7823.e27ea3dc-47a3-4885-aad6-a8b97b60b7d7',
    taskIds: ['7823.e27ea3dc-47a3-4885-aad6-a8b97b60b7d7'],
    totalBatchesProcessed: 1,
    totalCheckpointSize: '35',
    wallClockTime: 10.411985,
    warmStartCheckpointUuid: '',
  },
});

const mockExperiment = mapV1GetExperimentDetailsResponse({
  config: {
    bind_mounts: [],
    checkpoint_policy: 'best',
    checkpoint_storage: {
      access_key: null,
      bucket: 'det-determined-main-us-west-2-573932760021',
      endpoint_url: null,
      prefix: null,
      save_experiment_best: 0,
      save_trial_best: 1,
      save_trial_latest: 1,
      secret_key: null,
      type: 's3',
    },
    data: {},
    debug: false,
    description: '',
    entrypoint: 'python3 3_hpsearch.py',
    environment: {
      add_capabilities: [],
      drop_capabilities: [],
      environment_variables: {
        cpu: [],
        cuda: [],
        rocm: [],
      },
      force_pull_image: false,
      image: {
        cpu: 'determinedai/pytorch-ngc-dev:e960eae',
        cuda: 'determinedai/pytorch-ngc-dev:e960eae',
        rocm: 'determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-622d512',
      },
      pod_spec: null,
      ports: {},
      proxy_ports: [],
      registry_auth: null,
    },
    hyperparameters: {
      increment_by: {
        maxval: 8,
        minval: 2,
        type: 'int',
      },
    },
    integrations: null,
    labels: null,
    log_policies: [],
    max_restarts: 0,
    min_checkpoint_period: {
      batches: 0,
    },
    min_validation_period: {
      batches: 0,
    },
    name: 'core-api-stage-3-77777',
    optimizations: {
      aggregation_frequency: 1,
      auto_tune_tensor_fusion: false,
      average_aggregated_gradients: true,
      average_training_metrics: true,
      grad_updates_size_file: null,
      gradient_compression: false,
      mixed_precision: 'O0',
      tensor_fusion_cycle_time: 1,
      tensor_fusion_threshold: 64,
    },
    pbs: {},
    perform_initial_validation: false,
    profiling: {
      begin_on_batch: 0,
      enabled: false,
      end_after_batch: null,
      sync_timings: true,
    },
    project: 'Uncategorized',
    records_per_epoch: 0,
    reproducibility: {
      experiment_seed: 1718387140,
    },
    resources: {
      devices: [],
      is_single_node: null,
      max_slots: null,
      native_parallel: false,
      priority: null,
      resource_pool: 'compute-pool',
      shm_size: null,
      slots_per_trial: 1,
      weight: 1,
    },
    scheduling_unit: 100,
    searcher: {
      bracket_rungs: [],
      divisor: 4,
      max_concurrent_trials: 16,
      max_length: {
        batches: 1,
      },
      max_rungs: 5,
      max_trials: 10,
      metric: 'x',
      mode: 'standard',
      name: 'adaptive_asha',
      smaller_is_better: true,
      source_checkpoint_uuid: null,
      source_trial_id: null,
      stop_once: false,
    },
    slurm: {},
    workspace: 'Uncategorized',
  },
  experiment: {
    archived: false,
    checkpointCount: 10,
    checkpointSize: '350',
    config: {
      bind_mounts: [],
      checkpoint_policy: 'best',
      checkpoint_storage: {
        access_key: null,
        bucket: 'det-determined-main-us-west-2-573932760021',
        endpoint_url: null,
        prefix: null,
        save_experiment_best: 0,
        save_trial_best: 1,
        save_trial_latest: 1,
        secret_key: null,
        type: 's3',
      },
      data: {},
      debug: false,
      description: '',
      entrypoint: 'python3 3_hpsearch.py',
      environment: {
        add_capabilities: [],
        drop_capabilities: [],
        environment_variables: {
          cpu: [],
          cuda: [],
          rocm: [],
        },
        force_pull_image: false,
        image: {
          cpu: 'determinedai/pytorch-ngc-dev:e960eae',
          cuda: 'determinedai/pytorch-ngc-dev:e960eae',
          rocm: 'determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-622d512',
        },
        pod_spec: null,
        ports: {},
        proxy_ports: [],
        registry_auth: null,
      },
      hyperparameters: {
        increment_by: {
          maxval: 8,
          minval: 2,
          type: 'int',
        },
      },
      integrations: null,
      labels: null,
      log_policies: [],
      max_restarts: 0,
      min_checkpoint_period: {
        batches: 0,
      },
      min_validation_period: {
        batches: 0,
      },
      name: 'core-api-stage-3-77777',
      optimizations: {
        aggregation_frequency: 1,
        auto_tune_tensor_fusion: false,
        average_aggregated_gradients: true,
        average_training_metrics: true,
        grad_updates_size_file: null,
        gradient_compression: false,
        mixed_precision: 'O0',
        tensor_fusion_cycle_time: 1,
        tensor_fusion_threshold: 64,
      },
      pbs: {},
      perform_initial_validation: false,
      profiling: {
        begin_on_batch: 0,
        enabled: false,
        end_after_batch: null,
        sync_timings: true,
      },
      project: 'Uncategorized',
      records_per_epoch: 0,
      reproducibility: {
        experiment_seed: 1718387140,
      },
      resources: {
        devices: [],
        is_single_node: null,
        max_slots: null,
        native_parallel: false,
        priority: null,
        resource_pool: 'compute-pool',
        shm_size: null,
        slots_per_trial: 1,
        weight: 1,
      },
      scheduling_unit: 100,
      searcher: {
        bracket_rungs: [],
        divisor: 4,
        max_concurrent_trials: 16,
        max_length: {
          batches: 1,
        },
        max_rungs: 5,
        max_trials: 10,
        metric: 'x',
        mode: 'standard',
        name: 'adaptive_asha',
        smaller_is_better: true,
        source_checkpoint_uuid: null,
        source_trial_id: null,
        stop_once: false,
      },
      slurm: {},
      workspace: 'Uncategorized',
    },
    description: '',
    displayName: '',
    endTime: '2024-07-18T15:32:51.133795Z',
    forkedFrom: 7591,
    hyperparameters: null,
    id: 7823,
    jobId: '890929a2-659c-4139-bc07-bde3f9d1b43a',
    labels: [],
    modelDefinitionSize: 5539,
    name: 'core-api-stage-3-77777',
    notes: '#wow #woah\n\n### wow woah!!',
    numTrials: 10,
    originalConfig:
      'environment:\n  add_capabilities: []\n  drop_capabilities: []\n  environment_variables:\n    cpu: []\n    cuda: []\n    rocm: []\n  force_pull_image: false\n  image:\n    cpu: determinedai/pytorch-ngc-dev:e960eae\n    cuda: determinedai/pytorch-ngc-dev:e960eae\n    rocm: determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-622d512\n  pod_spec: null\n  ports: {}\n  proxy_ports: []\nproject: Uncategorized\nworkspace: Uncategorized\nbind_mounts: []\ncheckpoint_policy: best\ncheckpoint_storage:\n  access_key: null\n  bucket: det-determined-main-us-west-2-573932760021\n  endpoint_url: null\n  prefix: null\n  save_experiment_best: 0\n  save_trial_best: 1\n  save_trial_latest: 1\n  secret_key: null\n  type: s3\ndata: {}\ndebug: false\ndescription: null\nentrypoint: python3 3_hpsearch.py\nhyperparameters:\n  increment_by:\n    maxval: 8\n    minval: 2\n    type: int\nintegrations: null\nlabels: []\nlog_policies: []\nmax_restarts: 0\nmin_checkpoint_period:\n  batches: 0\nmin_validation_period:\n  batches: 0\nname: core-api-stage-3-77777\noptimizations:\n  aggregation_frequency: 1\n  auto_tune_tensor_fusion: false\n  average_aggregated_gradients: true\n  average_training_metrics: true\n  grad_updates_size_file: null\n  gradient_compression: false\n  mixed_precision: O0\n  tensor_fusion_cycle_time: 1\n  tensor_fusion_threshold: 64\npbs: {}\nperform_initial_validation: false\nprofiling:\n  begin_on_batch: 0\n  enabled: false\n  end_after_batch: null\n  sync_timings: true\nrecords_per_epoch: 0\nreproducibility:\n  experiment_seed: 1718387140\nresources:\n  devices: []\n  is_single_node: null\n  max_slots: null\n  native_parallel: false\n  priority: null\n  resource_pool: compute-pool\n  shm_size: null\n  slots_per_trial: 1\n  weight: 1\nscheduling_unit: 100\nsearcher:\n  bracket_rungs: []\n  divisor: 4\n  max_concurrent_trials: 16\n  max_length:\n    batches: 1\n  max_rungs: 5\n  max_trials: 10\n  metric: x\n  mode: standard\n  name: adaptive_asha\n  smaller_is_better: true\n  source_checkpoint_uuid: null\n  source_trial_id: null\n  stop_once: false\nslurm: {}\n',
    parentArchived: false,
    progress: 1,
    projectId: 1,
    projectName: 'Uncategorized',
    projectOwnerId: 1,
    resourcePool: 'compute-pool',
    searcherMetric: '',
    searcherType: '"adaptive_asha"',
    startTime: '2024-07-18T15:31:01.441050Z',
    state: 'STATE_COMPLETED',
    trialIds: [54170, 54171, 54172, 54173, 54174, 54175, 54176, 54177, 54178, 54179],
    unmanaged: false,
    userId: 595,
    username: 'ashton',
    workspaceId: 1,
    workspaceName: 'Uncategorized',
  },
});

const setupTest = (props: Partial<Props> = {}) => {
  const outerRef: { current: null | (() => void) } = { current: null };
  const Wrapper = () => {
    const { Component, open } = useModal(ExperimentContinueModalComponent);

    useLayoutEffect(() => {
      outerRef.current = open;
    });

    return (
      <ThemeProvider>
        <UIProvider theme={DefaultTheme.Light} themeIsDark>
          <Component
            experiment={mockExperiment}
            trial={mockTrial}
            type={ContinueExperimentType.Continue}
            {...props}
          />
        </UIProvider>
      </ThemeProvider>
    );
  };

  const container = render(<Wrapper />);

  return { container, openRef: outerRef };
};

describe('ExperimentContinueModal', () => {
  afterEach(() => {
    mockUseFeature.mockReset();
  });
  it('should render', () => {
    const { container, openRef } = setupTest();

    act(() => {
      openRef.current?.();
    });

    expect(container.queryByText('Continue Trial in New Experiment')).toBeInTheDocument();
  });

  it('should show proper copy when f_flat_runs is on', () => {
    mockUseFeature.mockReturnValue(true);
    const { container, openRef } = setupTest();

    act(() => {
      openRef.current?.();
    });

    expect(container.queryByText('Continue as New Run')).toBeInTheDocument();
  });
});
