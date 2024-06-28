import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';

import { handlePath } from 'routes/utils';
import {
  archiveExperiment,
  cancelExperiment,
  changeExperimentLogRetention,
  deleteExperiment,
  killExperiment,
  patchExperiment,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import { RunState } from 'types';

import ExperimentActionDropdown, { Action } from './ExperimentActionDropdown';
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
  cancelExperiment: vi.fn(),
  deleteExperiment: vi.fn(),
  getWorkspaces: vi.fn(() => Promise.resolve({ workspaces: [] })),
  killExperiment: vi.fn(),
  patchExperiment: vi.fn(),
  pauseExperiment: vi.fn(),
  unarchiveExperiment: vi.fn(),
}));

const mocks = vi.hoisted(() => {
  return {
    canCreateExperiment: vi.fn(),
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
      canCreateExperiment: mocks.canCreateExperiment,
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
    mocks.canDeleteExperiment.mockImplementation(() => true);
    setup();
    await user.click(screen.getByText(Action.Delete));
    await user.click(screen.getByRole('button', { name: Action.Delete }));
    expect(vi.mocked(deleteExperiment)).toBeCalled();
  });

  it('should hide Delete option without permissions', () => {
    mocks.canDeleteExperiment.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Delete)).not.toBeInTheDocument();
  });

  it('should provide Kill option', async () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup(undefined, RunState.Paused, undefined);
    await user.click(screen.getByText(Action.Kill));
    await user.click(screen.getByRole('button', { name: Action.Kill }));
    expect(vi.mocked(killExperiment)).toBeCalled();
  });

  it('should hide Kill option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup(undefined, RunState.Paused, undefined);
    expect(screen.queryByText(Action.Kill)).not.toBeInTheDocument();
  });

  it('should provide Archive option', async () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup();
    await user.click(screen.getByText(Action.Archive));
    expect(vi.mocked(archiveExperiment)).toBeCalled();
  });

  it('should hide Archive option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Archive)).not.toBeInTheDocument();
  });

  it('should provide Unarchive option', async () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup(undefined, undefined, true);
    await user.click(screen.getByText(Action.Unarchive));
    expect(vi.mocked(unarchiveExperiment)).toBeCalled();
  });

  it('should hide Unarchive option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup(undefined, undefined, true);
    expect(screen.queryByText(Action.Unarchive)).not.toBeInTheDocument();
  });

  it('should provide Move option', () => {
    mocks.canMoveExperiment.mockImplementation(() => true);
    setup();
    expect(screen.getByText(Action.Move)).toBeInTheDocument();
  });

  it('should hide Move option without permissions', () => {
    mocks.canMoveExperiment.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Move)).not.toBeInTheDocument();
  });

  it('should provide Edit option', async () => {
    mocks.canModifyExperimentMetadata.mockImplementation(() => true);
    setup();
    await user.click(screen.getByText(Action.Edit));
    await user.type(screen.getByRole('textbox', { name: 'Name' }), 'edit');
    await user.click(screen.getByText('Save'));
    expect(vi.mocked(patchExperiment)).toBeCalled();
  });

  it('should hide Edit option without permissions', () => {
    mocks.canModifyExperimentMetadata.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.Edit)).not.toBeInTheDocument();
  });

  it('should provide Pause option', async () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup(undefined, RunState.Running);
    await user.click(screen.getByText(Action.Pause));
    expect(vi.mocked(pauseExperiment)).toBeCalled();
  });

  it('should hide Pause option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup(undefined, RunState.Running);
    expect(screen.queryByText(Action.Pause)).not.toBeInTheDocument();
  });

  it('should provide Cancel option', async () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup(undefined, RunState.Running);
    await user.click(screen.getByText(Action.Cancel));
    expect(vi.mocked(cancelExperiment)).toBeCalled();
  });

  it('should hide Cancel option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup(undefined, RunState.Running);
    expect(screen.queryByText(Action.Cancel)).not.toBeInTheDocument();
  });

  it('should provide Retain Logs option', () => {
    mocks.canModifyExperiment.mockImplementation(() => true);
    setup();
    expect(screen.queryByText(Action.RetainLogs)).toBeInTheDocument();
  });

  it('should hide Retain Logs option without permissions', () => {
    mocks.canModifyExperiment.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.RetainLogs)).not.toBeInTheDocument();
  });

  it('should provide Tensor Board option', () => {
    mocks.canViewExperimentArtifacts.mockImplementation(() => true);
    setup();
    expect(screen.getByText(Action.OpenTensorBoard)).toBeInTheDocument();
  });

  it('should hide Tensor Board option without permissions', () => {
    mocks.canViewExperimentArtifacts.mockImplementation(() => false);
    setup();
    expect(screen.queryByText(Action.OpenTensorBoard)).not.toBeInTheDocument();
  });
});
