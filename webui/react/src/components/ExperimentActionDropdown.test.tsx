import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';

import { handlePath } from 'routes/utils';
import {
  archiveExperiment,
  deleteExperiment,
  killExperiment,
  unarchiveExperiment,
} from 'services/api';
import { RunState } from 'types';

import ExperimentActionDropdown from './ExperimentActionDropdown';
import { cell, experiment } from './ExperimentActionDropdown.test.mock';

const user = userEvent.setup();

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
  archiveExperiment: vi.fn(),
  deleteExperiment: vi.fn(),
  getWorkspaces: vi.fn(() => Promise.resolve({ workspaces: [] })),
  killExperiment: vi.fn(),
  unarchiveExperiment: vi.fn(),
}));

const mocks = vi.hoisted(() => {
  return {
    canDeleteExperiment: vi.fn(),
    canModifyExperiment: vi.fn(),
    canModifyExperimentMetadata: vi.fn(),
    canMoveExperiment: vi.fn(),
    canViewExperimentArtifacts: vi.fn(),
  };
});

vi.mock('hooks/usePermissions', () => {
  const usePermissions = vi.fn(() => {
    return {
      canDeleteExperiment: mocks.canDeleteExperiment,
      canModifyExperiment: mocks.canModifyExperiment,
      canModifyExperimentMetadata: mocks.canModifyExperimentMetadata,
      canMoveExperiment: mocks.canMoveExperiment,
      canViewExperimentArtifacts: mocks.canViewExperimentArtifacts,
    };
  });
  return {
    default: usePermissions,
  };
});

const setup = (link?: string, state?: RunState, archived?: boolean) => {
  const onComplete = vi.fn();
  const onVisibleChange = vi.fn();
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ConfirmationProvider>
        <ExperimentActionDropdown
          cell={cell}
          experiment={{
            ...experiment,
            archived: archived === undefined ? experiment.archived : archived,
            state: state === undefined ? experiment.state : state,
          }}
          isContextMenu
          link={link}
          makeOpen
          onComplete={onComplete}
          onVisibleChange={onVisibleChange}>
          <div />
        </ExperimentActionDropdown>
      </ConfirmationProvider>
    </UIProvider>,
  );
  return {
    onComplete,
    onVisibleChange,
  };
};

describe('ExperimentActionDropdown', () => {
  it('should provide Copy Data option', async () => {
    setup();
    mockNavigatorClipboard();
    await user.click(screen.getByText('Copy Value'));
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(cell.copyData);
  });

  it('should provide Link option', async () => {
    const link = 'https://www.google.com/';
    setup(link);
    await user.click(screen.getByText('Open Link in New Tab'));
    const tabClick = vi.mocked(handlePath).mock.calls[0];
    expect(tabClick[0]).toMatchObject({ type: 'click' });
    expect(tabClick[1]).toMatchObject({
      path: link,
      popout: 'tab',
    });
    await user.click(screen.getByText('Open Link in New Window'));
    const windowClick = vi.mocked(handlePath).mock.calls[1];
    expect(windowClick[0]).toMatchObject({ type: 'click' });
    expect(windowClick[1]).toMatchObject({
      path: link,
      popout: 'window',
    });
  });

  it('should provide Delete option', async () => {
    mocks.canDeleteExperiment.mockImplementation(() => true);
    setup();
    await user.click(screen.getByText('Delete'));
    await user.click(screen.getByRole('button', { name: 'Delete' }));
    expect(vi.mocked(deleteExperiment)).toBeCalled();
  });

  it('should hide Delete option without permissions', () => {
    mocks.canDeleteExperiment.mockImplementation(() => false);
    setup();
    expect(screen.queryByText('Delete')).not.toBeInTheDocument();
  });

  it('should provide Kill option', async () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup(undefined, RunState.Paused, undefined);
    await user.click(screen.getByText('Kill'));
    await user.click(screen.getByRole('button', { name: 'Kill' }));
    expect(vi.mocked(killExperiment)).toBeCalled();
  });

  it('should hide Kill option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup(undefined, RunState.Paused, undefined);
    expect(screen.queryByText('Kill')).not.toBeInTheDocument();
  });

  it('should provide Archive option', async () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup();
    await user.click(screen.getByText('Archive'));
    expect(vi.mocked(archiveExperiment)).toBeCalled();
  });

  it('should hide Archive option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup();
    expect(screen.queryByText('Archive')).not.toBeInTheDocument();
  });

  it('should provide Unarchive option', async () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup(undefined, undefined, true);
    await user.click(screen.getByText('Unarchive'));
    expect(vi.mocked(unarchiveExperiment)).toBeCalled();
  });

  it('should hide Unarchive option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup(undefined, undefined, true);
    expect(screen.queryByText('Unarchive')).not.toBeInTheDocument();
  });

  it('should provide Move option', () => {
    mocks.canMoveExperiment.mockImplementation(() => true);
    setup();
    expect(screen.getByText('Move')).toBeInTheDocument();
  });

  it('should hide Move option without permissions', () => {
    mocks.canMoveExperiment.mockImplementation(() => false);
    setup();
    expect(screen.queryByText('Move')).not.toBeInTheDocument();
  });
});
