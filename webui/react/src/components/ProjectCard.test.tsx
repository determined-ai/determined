import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';

import { handlePath } from 'routes/utils';
import { Project } from 'types';

import ProjectCard from './ProjectCard';
import { ThemeProvider } from './ThemeProvider';

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
  numRuns: 16,
  state: 'UNSPECIFIED',
  userId: 1354,
  workspaceId: 1684,
  workspaceName: '',
};

const user = userEvent.setup();

const setup = () => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <ProjectCard project={projectMock} />
      </ThemeProvider>
    </UIProvider>,
  );
};

vi.mock('routes/utils', () => ({
  handlePath: vi.fn(),
  paths: {
    projectDetails: () => 'testPath',
  },
  serverAddress: () => 'http://localhost',
}));

describe('ProjectCard', () => {
  it('should display project name', () => {
    setup();
    expect(screen.getByText(projectMock.name)).toBeInTheDocument();
  });

  it('should display experiments count', () => {
    setup();
    expect(
      screen.getByText(projectMock.numRuns?.toString() ?? 'Count undefined'),
    ).toBeInTheDocument();
  });

  it('should display archived label', () => {
    setup();
    expect(screen.queryByText('Archived')).not.toBeInTheDocument();
    projectMock.archived = true;
    setup();
    expect(screen.getByText('Archived')).toBeInTheDocument();
  });

  it('should navigate to details page on click', async () => {
    setup();
    await user.click(screen.getByTestId(`card-${projectMock.name}`));
    const click = vi.mocked(handlePath).mock.calls[0];
    expect(click[0]).toMatchObject({ type: 'click' });
    expect(click[1]).toMatchObject({
      path: 'testPath',
    });
  });
});
