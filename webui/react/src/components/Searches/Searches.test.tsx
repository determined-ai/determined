import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';
import { Project } from 'types';

import Searches from './Searches';
import { defaultProjectSettings } from './Searches.settings';

const projectMock: Project = {
  archived: false,
  description: '',
  id: 1849,
  immutable: false,
  lastExperimentStartedAt: '2024-06-03T19:33:38.731220Z',
  name: 'test',
  notes: [],
  numActiveExperiments: 1,
  numExperiments: 16,
  state: 'UNSPECIFIED',
  userId: 1354,
  workspaceId: 1684,
  workspaceName: '',
};

const expectedFilterString = JSON.stringify({
  filterGroup: {
    children: [
      { children: [], conjunction: 'and', kind: 'group' },
      {
        columnName: 'searcherType',
        kind: 'field',
        location: 'LOCATION_TYPE_EXPERIMENT',
        operator: '!=',
        type: 'COLUMN_TYPE_TEXT',
        value: 'single',
      },
    ],
    conjunction: 'and',
    kind: 'group',
  },
  showArchived: false,
});

const searchExperimentsMock = vi.hoisted(() =>
  vi.fn().mockReturnValue(
    Promise.resolve({
      experiments: [],
      pagination: { total: 0 },
    }),
  ),
);

vi.mock('services/api', () => ({
  getProjectColumns: vi.fn().mockReturnValue([]),
  getWorkspaces: vi.fn().mockResolvedValue({ workspaces: [] }),
  resetUserSetting: () => Promise.resolve(),
  searchExperiments: searchExperimentsMock,
}));

vi.mock('stores/userSettings', async (importOriginal) => {
  const userSettings = await import('stores/userSettings');
  const store = new userSettings.UserSettingsStore();

  store.clear();

  return {
    ...(await importOriginal<typeof import('stores/userSettings')>()),
    default: store,
  };
});

vi.mock('hooks/useMobile', async (importOriginal) => {
  return {
    ...(await importOriginal<typeof import('hooks/useMobile')>()),
    default: () => false,
  };
});

const user = userEvent.setup();

const setup = () => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ConfirmationProvider>
        <ThemeProvider>
          <HelmetProvider>
            <BrowserRouter>
              <Searches project={projectMock} />
            </BrowserRouter>
          </HelmetProvider>
        </ThemeProvider>
      </ConfirmationProvider>
    </UIProvider>,
  );
};

describe('Searches', () => {
  it('should display with correct label', () => {
    setup();

    expect(screen.getByText('Loading searches...')).toBeInTheDocument();
  });

  it('should display column picker menu without tab selection', async () => {
    setup();

    await user.click(screen.getByTestId('columns-menu-button'));
    expect(screen.queryByRole('tab')).not.toBeInTheDocument();
    expect(screen.getByTestId('column-picker-tab')).toBeInTheDocument();
  });

  it('should have hidden filter to exclude single-trial experiments', () => {
    setup();

    expect(vi.mocked(searchExperimentsMock)).toHaveBeenCalledWith(
      expect.objectContaining({
        filter: expectedFilterString,
        limit: defaultProjectSettings.pageLimit,
        offset: 0,
        projectId: projectMock.id,
        sort: defaultProjectSettings.sortString,
      }),
      { signal: expect.any(AbortSignal) },
    );
  });
});
