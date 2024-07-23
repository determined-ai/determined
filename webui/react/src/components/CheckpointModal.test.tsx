import { act, render } from '@testing-library/react';
import { useModal } from 'hew/Modal';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';
import { useLayoutEffect } from 'react';

import { CheckpointState } from 'types';

import CheckpointModalComponent, { Props } from './CheckpointModal';
import { ThemeProvider } from './ThemeProvider';

const mockUseFeature = vi.hoisted(() => vi.fn());
vi.mock('hooks/useFeature', () => {
  return {
    default: () => ({
      isOn: mockUseFeature,
    }),
  };
});

const mockConfig = {
  checkpointPolicy: 'best',
  checkpointStorage: {
    bucket: 'det-determined-master-us-west-2-573932760021',
    saveExperimentBest: 0,
    saveTrialBest: 1,
    saveTrialLatest: 1,
    type: 's3' as const,
  },
  hyperparameters: {
    global_batch_size: { type: 'const' as const, val: 64 },
    hidden_size: { type: 'const' as const, val: 64 },
    learning_rate: { maxval: 0.1, minval: 0.0001, type: 'double' as const },
  },
  labels: [],
  maxRestarts: 5,
  name: 'mnist_pytorch_lightning_adaptive',
  profiling: { enabled: false },
  resources: {},
  searcher: {
    max_length: { batches: 937, epochs: 1, records: 1 },
    max_trials: 16,
    metric: 'val_loss',
    name: 'adaptive_asha' as const,
    smallerIsBetter: true,
  },
};

const mockCheckpoint = {
  endTime: '2022-07-20T19:58:58.441283Z',
  experimentId: 100,
  metadata: {
    determined_version: '0.18.5-dev0',
    format: 'pickle',
    framework: 'torch-1.10.2+cu113',
  },
  resources: {
    'code/': 0,
    'code/.ipynb_checkpoints/': 0,
    'code/adaptive.yaml': 1067,
    'code/adaptive-fast.yaml': 1099,
    'code/adsf.yaml': 1047,
    'code/bonst.yaml': 434,
    'code/checkpoints/': 0,
    'code/checkpoints/0669b753-fcea-4ccd-b894-18a23d538e27/': 0,
    'code/const.yaml': 1063,
    'code/data.py': 1449,
    'code/distributed.yaml': 499,
    'code/layers.py': 568,
    'code/model_def.py': 3745,
    'code/prof.yaml': 1865,
    'code/README.md': 1407,
    'code/tmp/': 0,
    'code/tmp/MNIST/': 0,
    'code/tmp/MNIST/processed/': 0,
    'code/tmp/MNIST/processed/test.pt': 7920407,
    'code/tmp/MNIST/processed/training.pt': 47520407,
    'code/tmp/MNIST/pytorch_mnist.tar.gz': 11613630,
    'load_data.json': 2882,
    'state_dict.pth': 15936663,
    'workload_sequencer.pkl': 89,
  },
  state: CheckpointState.Completed,
  totalBatches: 1,
  trialId: 200,
  uuid: '90d58796-f746-48fb-bf1b-b7761ad9314d',
};

const TITLE = 'MODAL TITLE';

const setupTest = (props: Partial<Props> = {}) => {
  const outerRef: { current: null | (() => void) } = { current: null };
  const Wrapper = () => {
    const { Component, open } = useModal(CheckpointModalComponent);

    useLayoutEffect(() => {
      outerRef.current = open;
    });

    return (
      <ThemeProvider>
        <UIProvider theme={DefaultTheme.Light} themeIsDark>
          <ConfirmationProvider>
            <Component checkpoint={mockCheckpoint} config={mockConfig} title={TITLE} {...props} />
          </ConfirmationProvider>
        </UIProvider>
      </ThemeProvider>
    );
  };

  const container = render(<Wrapper />);

  return { container, openRef: outerRef };
};

describe('CheckpointModal', () => {
  afterEach(() => {
    mockUseFeature.mockReset();
  });
  it('shows the component when open', () => {
    const { container, openRef } = setupTest();

    act(() => {
      openRef.current?.();
    });

    expect(container.queryByText(TITLE)).toBeInTheDocument();
    expect(container.queryByText(TITLE)).toBeVisible();
  });

  it('shows run copy when runs flag is on', () => {
    mockUseFeature.mockReturnValue(true);
    const { container, openRef } = setupTest();

    act(() => {
      openRef.current?.();
    });
    expect(container.queryByText('Search 100')).toBeInTheDocument();
    expect(container.queryByText('Search 100')).toBeVisible();
    expect(container.queryByText('Run 200')).toBeInTheDocument();
    expect(container.queryByText('Run 200')).toBeVisible();
  });
});
