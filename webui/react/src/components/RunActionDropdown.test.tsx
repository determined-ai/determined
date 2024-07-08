import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';

import { handlePath } from 'routes/utils';
import { archiveRuns, deleteRuns, killRuns, unarchiveRuns } from 'services/api';
import { FlatRunExperiment, RunState } from 'types';

import RunActionDropdown, { Action } from './RunActionDropdown';
import { cell, run } from './RunActionDropdown.test.mock';

const mockNavigatorClipboard = () => {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: {
      readText: vi.fn(),
      writeText: vi.fn(),
    },
    writable: true,
  });
};

vi.mock('routes/utils', () => ({
  handlePath: vi.fn(),
  serverAddress: () => 'http://localhost',
}));

vi.mock('services/api', () => ({
  archiveRuns: vi.fn(),
  deleteRuns: vi.fn(),
  killRuns: vi.fn(),
  unarchiveRuns: vi.fn(),
}));

const mocks = vi.hoisted(() => {
  return {
    canDeleteFlatRun: vi.fn(),
    canModifyFlatRun: vi.fn(),
    canMoveFlatRun: vi.fn(),
  };
});

vi.mock('hooks/usePermissions', () => {
  const usePermissions = vi.fn(() => {
    return {
      canDeleteFlatRun: mocks.canDeleteFlatRun,
      canModifyFlatRun: mocks.canModifyFlatRun,
      canMoveFlatRun: mocks.canMoveFlatRun,
    };
  });
  return {
    default: usePermissions,
  };
});

const setup = (
  link?: string,
  state?: RunState,
  archived?: boolean,
  experiment?: FlatRunExperiment,
) => {
  const onComplete = vi.fn();
  const onVisibleChange = vi.fn();
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ConfirmationProvider>
        <RunActionDropdown
          cell={cell}
          link={link}
          makeOpen
          projectId={run.projectId}
          run={{
            ...run,
            archived: archived === undefined ? run.archived : archived,
            experiment: experiment === undefined ? run.experiment : experiment,
            state: state === undefined ? run.state : state,
          }}
          onComplete={onComplete}
          onVisibleChange={onVisibleChange}
        />
      </ConfirmationProvider>
    </UIProvider>,
  );
  return {
    onComplete,
    onVisibleChange,
  };
};

const user = userEvent.setup();

describe('RunActionDropdown', () => {
  it('should provide Copy Data option', async () => {
    setup();
    mockNavigatorClipboard();
    await user.click(screen.getByText(Action.Copy));
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(cell.copyData);
  });

  it('should provide Link option', async () => {
    const link = 'https://www.google.com/';
    setup(link);
    await user.click(screen.getByText(Action.NewTab));
    const tabClick = vi.mocked(handlePath).mock.calls[0];
    expect(tabClick[0]).toMatchObject({ type: 'click' });
    expect(tabClick[1]).toMatchObject({
      path: link,
      popout: 'tab',
    });
    await user.click(screen.getByText(Action.NewWindow));
    const windowClick = vi.mocked(handlePath).mock.calls[1];
    expect(windowClick[0]).toMatchObject({ type: 'click' });
    expect(windowClick[1]).toMatchObject({
      path: link,
      popout: 'window',
    });
  });

  it('should provide Delete option', async () => {
    mocks.canDeleteFlatRun.mockImplementation(() => true);
    setup();
    await user.click(screen.getByText(Action.Delete));
    await user.click(screen.getByRole('button', { name: Action.Delete }));
    expect(vi.mocked(deleteRuns)).toBeCalled();
  });

  it('should hide Delete option without permissions', () => {
    mocks.canDeleteFlatRun.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Delete)).not.toBeInTheDocument();
  });

  it('should provide Kill option', async () => {
    mocks.canModifyFlatRun.mockImplementation(() => true);
    setup(undefined, RunState.Paused, undefined);
    await user.click(screen.getByText(Action.Kill));
    await user.click(screen.getByRole('button', { name: Action.Kill }));
    expect(vi.mocked(killRuns)).toBeCalled();
  });

  it('should hide Kill option without permissions', () => {
    mocks.canModifyFlatRun.mockImplementation(() => false);
    setup(undefined, RunState.Paused, undefined);
    expect(screen.queryByText(Action.Kill)).not.toBeInTheDocument();
  });

  it('should provide Archive option', async () => {
    mocks.canModifyFlatRun.mockImplementation(() => true);
    setup();
    await user.click(screen.getByText(Action.Archive));
    expect(vi.mocked(archiveRuns)).toBeCalled();
  });

  it('should hide Archive option without permissions', () => {
    mocks.canModifyFlatRun.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Archive)).not.toBeInTheDocument();
  });

  it('should provide Unarchive option', async () => {
    mocks.canModifyFlatRun.mockImplementation(() => true);
    setup(undefined, undefined, true);
    await user.click(screen.getByText(Action.Unarchive));
    expect(vi.mocked(unarchiveRuns)).toBeCalled();
  });

  it('should hide Unarchive option without permissions', () => {
    mocks.canModifyFlatRun.mockImplementation(() => false);
    setup(undefined, undefined, true);
    expect(screen.queryByText(Action.Unarchive)).not.toBeInTheDocument();
  });

  it('should provide Move option', () => {
    mocks.canMoveFlatRun.mockImplementation(() => true);
    setup();
    expect(screen.getByText(Action.Move)).toBeInTheDocument();
  });

  it('should hide Move option without permissions', () => {
    mocks.canMoveFlatRun.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Move)).not.toBeInTheDocument();
  });

  it('should provide Pause option', () => {
    mocks.canModifyFlatRun.mockImplementation(() => true);
    const experiment: FlatRunExperiment = {
      description: '',
      forkedFrom: 6634,
      id: 6833,
      isMultitrial: false,
      name: 'iris_tf_keras_adaptive_search',
      progress: 0.9444444,
      resourcePool: 'compute-pool',
      searcherMetric: 'val_categorical_accuracy',
      searcherType: 'single',
      unmanaged: false,
    };
    setup(undefined, RunState.Active, false, experiment);
    expect(screen.getByText(Action.Pause)).toBeInTheDocument();
  });

  it('should hide Pause option without permissions', () => {
    mocks.canModifyFlatRun.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Pause)).not.toBeInTheDocument();
  });

  it('should provide Resume option', () => {
    mocks.canModifyFlatRun.mockImplementation(() => true);
    setup(undefined, RunState.Paused, false);
    expect(screen.getByText(Action.Resume)).toBeInTheDocument();
  });

  it('should hide Resume option without permissions', () => {
    mocks.canModifyFlatRun.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Resume)).not.toBeInTheDocument();
  });
});
