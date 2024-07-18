import { render, screen, waitFor } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { useEffect } from 'react';
import { HelmetProvider } from 'react-helmet-async';

import { ThemeProvider } from 'components/ThemeProvider';
import authStore from 'stores/auth';
import userStore from 'stores/users';
import { DetailedUser } from 'types';

import Dashboard from './Dashboard';

const mocks = vi.hoisted(() => {
  return {
    getExperiments: vi.fn().mockImplementation(() =>
      Promise.resolve({
        experiments: [],
      }),
    ),
    getJupyterLabs: vi.fn().mockImplementation(() => Promise.resolve([])),
    getProjectsByUserActivity: vi.fn().mockImplementation(() => Promise.resolve([])),
  };
});

vi.mock('services/api', () => ({
  getCommands: () => Promise.resolve([]),
  getExperiments: mocks.getExperiments,
  getJupyterLabs: mocks.getJupyterLabs,
  getProjectsByUserActivity: mocks.getProjectsByUserActivity,
  getShells: () => Promise.resolve([]),
  getTensorBoards: () => Promise.resolve([]),
}));

const CURRENT_USER: DetailedUser = { id: 1, isActive: true, isAdmin: false, username: 'bunny' };

// for JupyterLabButton:
vi.mock('hooks/useSettings', async (importOriginal) => {
  const useSettings = vi.fn(() => {
    const settings = {
      jupyterLab: {
        alt: false,
        ctrl: false,
        key: 'L',
        meta: true,
        shift: true,
      },
    };
    return { isLoading: false, settings };
  });

  return {
    __esModule: true,
    ...(await importOriginal<typeof import('hooks/useSettings')>()),
    useSettings,
  };
});

vi.mock('hooks/usePermissions', () => {
  const usePermissions = vi.fn(() => {
    return {
      canCreateNSC: false,
    };
  });
  return {
    default: usePermissions,
  };
});

const Container: React.FC = () => {
  useEffect(() => {
    authStore.setAuth({ isAuthenticated: true });
    authStore.setAuthChecked();
    userStore.updateCurrentUser(CURRENT_USER);
  }, []);

  return <Dashboard />;
};

const setup = () => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <HelmetProvider>
          <Container />
        </HelmetProvider>
      </ThemeProvider>
    </UIProvider>,
  );
};

describe('Dashboard', () => {
  it('renders empty state', async () => {
    setup();

    await waitFor(() => {
      expect(screen.getByText('Your Recent Submissions')).toBeInTheDocument();
      expect(screen.getByText('No submissions')).toBeInTheDocument();
      expect(
        screen.getByText('Your recent experiments and tasks will show up here.'),
      ).toBeInTheDocument();
      expect(screen.getByText('Get started')).toBeInTheDocument();
      expect(screen.queryByText('Recently Viewed Projects')).not.toBeInTheDocument();
    });
  });

  it('renders ProjectCards with project data', async () => {
    mocks.getProjectsByUserActivity.mockImplementation(() =>
      Promise.resolve([
        {
          archived: false,
          description: '',
          errorMessage: '',
          id: 1,
          immutable: true,
          key: '',
          lastExperimentStartedAt: '2024-07-17T16:18:56.813686Z',
          name: 'Uncategorized',
          notes: [
            {
              contents: '',
              name: 'Untitled',
            },
          ],
          numActiveExperiments: 0,
          numExperiments: 1297,
          numRuns: 41995,
          state: 'UNSPECIFIED',
          userId: 1,
          username: 'admin',
          workspaceId: 1,
          workspaceName: 'Uncategorized',
        },
      ]),
    );

    setup();

    await waitFor(() => {
      expect(screen.queryByText('Recently Viewed Projects')).toBeInTheDocument();
      expect(screen.getByText('Uncategorized')).toBeInTheDocument();
    });
  });
  it('renders submissions', async () => {
    mocks.getJupyterLabs.mockImplementation(() =>
      Promise.resolve([
        {
          displayName: 'Test',
          id: 'dbf259fa-611a-4940-85c5-92463b289835',
          name: 'JupyterLab (eminently-moving-oryx)',
          resourcePool: 'aux-pool',
          serviceAddress:
            '/proxy/dbf259fa-611a-4940-85c5-92463b289835/?token=v2.public.eyJpZCI6MCwidGFza19pZCI6ImRiZjI1OWZhLTYxMWEtNDk0MC04NWM1LTkyNDYzYjI4OTgzNSIsInVzZXJfc2Vzc2lvbl9pZCI6bnVsbCwidXNlcl9pZCI6MTM1NH2nxyzr1aMT2lECxRF3FQDUW6I3O5EYY62JJSP_hcaiDCcRZN5EML1csdVZm76xXer8hsia0osi1DqNqmqW6eUG.bnVsbA',
          startTime: '2024-07-18T15:13:07.401Z',
          state: 'RUNNING',
          type: 'jupyter-lab',
          userId: 1354,
          workspaceId: 1,
        },
      ]),
    );
    mocks.getExperiments.mockImplementation(() =>
      Promise.resolve({
        experiments: [
          {
            archived: false,
            checkpoints: 0,
            checkpointSize: 0,
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
              entrypoint: 'python3 -m determined.launch.torch_distributed python3 train.py',
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
                  cpu: 'determinedai/pytorch-tensorflow-cpu-dev:8b3bea3',
                  cuda: 'determinedai/pytorch-tensorflow-cuda-dev:8b3bea3',
                  rocm: 'determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-622d512',
                },
                pod_spec: null,
                ports: {},
                proxy_ports: [],
                registry_auth: null,
              },
              hyperparameters: {
                dropout1: {
                  maxval: 0.8,
                  minval: 0.2,
                  type: 'double',
                },
                dropout2: {
                  maxval: 0.8,
                  minval: 0.2,
                  type: 'double',
                },
                learning_rate: {
                  maxval: 2,
                  minval: 0.0002,
                  type: 'double',
                },
                n_filters1: {
                  maxval: 64,
                  minval: 8,
                  type: 'int',
                },
                n_filters2: {
                  maxval: 72,
                  minval: 8,
                  type: 'int',
                },
              },
              integrations: null,
              labels: [],
              log_policies: [],
              max_restarts: 5,
              min_checkpoint_period: {
                batches: 0,
              },
              min_validation_period: {
                batches: 0,
              },
              name: 'mnist_pytorch_dist_random_search',
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
                experiment_seed: 1717172336,
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
                max_concurrent_trials: 16,
                max_length: {
                  epochs: 1,
                },
                max_trials: 2,
                metric: 'accuracy',
                name: 'random',
                smaller_is_better: true,
                source_checkpoint_uuid: null,
                source_trial_id: null,
              },
              slurm: {},
              workspace: 'Uncategorized',
            },
            description: '',
            duration: 35,
            endTime: '2024-06-14T15:51:34.475103Z',
            forkedFrom: 6832,
            hyperparameters: {
              dropout1: {
                maxval: 0.8,
                minval: 0.2,
                type: 'double',
              },
              dropout2: {
                maxval: 0.8,
                minval: 0.2,
                type: 'double',
              },
              learning_rate: {
                maxval: 2,
                minval: 0.0002,
                type: 'double',
              },
              n_filters1: {
                maxval: 64,
                minval: 8,
                type: 'int',
              },
              n_filters2: {
                maxval: 72,
                minval: 8,
                type: 'int',
              },
            },
            id: 6936,
            jobId: '7a1f1e82-04ca-4e99-88ca-2a62230ca064',
            labels: [],
            name: 'mnist_pytorch_dist_random_search',
            notes: '',
            numTrials: 2,
            progress: 0,
            projectId: 1,
            projectName: 'Uncategorized',
            resourcePool: 'compute-pool',
            searcherType: 'random',
            startTime: '2024-06-14T15:50:59.059559Z',
            state: 'CANCELED',
            trialIds: [],
            unmanaged: false,
            userId: 1354,
            workspaceId: 1,
            workspaceName: 'Uncategorized',
          },
        ],
      }),
    );

    setup();

    await waitFor(() => {
      expect(screen.getByText('JupyterLab (eminently-moving-oryx)')).toBeInTheDocument();
      expect(screen.getByText('mnist_pytorch_dist_random_search')).toBeInTheDocument();
    });
  });
});
