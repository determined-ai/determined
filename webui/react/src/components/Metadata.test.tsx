import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { isArray } from 'lodash';

import { JsonObject, TrialDetails } from 'types';
import { downloadText } from 'utils/browser';
import { isJsonObject } from 'utils/data';

import Metadata, { EMPTY_MESSAGE } from './Metadata';
import { ThemeProvider } from './ThemeProvider';

const mockMetadata = {
  other: {
    test: 105,
    testing: 'asdf',
  },
  steps_completed: 101,
};

const mockTrial: TrialDetails = {
  autoRestarts: 0,
  bestAvailableCheckpoint: {
    endTime: '2024-07-09T18:11:37.179665Z',
    resources: {
      'metadata.json': 28,
      'state': 13,
    },
    state: 'COMPLETED',
    totalBatches: 100,
    uuid: 'd1f2ea2f-4872-4b3d-a6ba-171647d87d49',
  },
  bestValidationMetric: {
    endTime: '2024-07-09T18:11:37.428948Z',
    metrics: {
      x: 100,
    },
    totalBatches: 100,
  },
  checkpointCount: 2,
  endTime: '2024-07-09T18:11:51.633537Z',
  experimentId: 7525,
  hyperparameters: {},
  id: 51646,
  latestValidationMetric: {
    endTime: '2024-07-09T18:11:37.428948Z',
    metrics: {
      x: 100,
    },
    totalBatches: 100,
  },
  metadata: mockMetadata,
  searcherMetricsVal: 0,
  startTime: '2024-07-09T18:05:42.629537Z',
  state: 'COMPLETED',
  summaryMetrics: {
    avgMetrics: {
      x: {
        count: 10,
        last: 100,
        max: 100,
        min: 10,
        sum: 550,
        type: 'number',
      },
    },
    validationMetrics: {
      x: {
        count: 1,
        last: 100,
        max: 100,
        min: 100,
        sum: 100,
        type: 'number',
      },
    },
  },
  taskId: '7525.048f395e-b68a-43f1-9a9f-e63720eb98be',
  totalBatchesProcessed: 100,
  totalCheckpointSize: 79,
};

vi.mock('utils/browser', () => ({
  downloadText: vi.fn(),
}));

const user = userEvent.setup();

const setup = (empty?: boolean) => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <Metadata trial={empty ? undefined : mockTrial} />,
      </ThemeProvider>
    </UIProvider>,
  );
};

describe('Metadata', () => {
  it('should display empty state', () => {
    setup(true);
    expect(screen.getByText(EMPTY_MESSAGE)).toBeInTheDocument();
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('should allow metadata download', async () => {
    setup();
    await user.click(screen.getByRole('button'));
    expect(vi.mocked(downloadText)).toBeCalledWith(`${mockTrial?.id}_metadata.json`, [
      JSON.stringify(mockTrial.metadata),
    ]);
  });

  it('should display Tree with metadata', () => {
    setup();
    const treeValues: string[] = [];
    const extractTreeValuesFromObject = (object: JsonObject) => {
      for (const [key, value] of Object.entries(object)) {
        if (value === null) continue;
        if (isJsonObject(value)) {
          extractTreeValuesFromObject(value);
          treeValues.push(key);
        } else {
          let stringValue = '';
          if (isArray(value)) {
            stringValue = `[${value.join(', ')}]`;
          } else {
            stringValue = value.toString();
          }
          treeValues.push(`${key}:`);
          treeValues.push(stringValue);
        }
      }
    };
    extractTreeValuesFromObject(mockMetadata);
    treeValues.forEach((treeValue) => {
      expect(screen.getByText(treeValue.toString())).toBeInTheDocument();
    });
  });
});
