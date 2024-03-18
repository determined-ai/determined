import { waitFor } from '@testing-library/react';
import { NotLoaded } from 'hew/utils/loadable';
import { List } from 'immutable';

import { Project } from 'types';

import { ProjectStore } from './projects';

vi.mock('services/api', () => ({
  getWorkspaceProjects: vi.fn(() =>
    Promise.resolve({ projects: [{ id: 1, name: 'project_1', workspaceId: 1 }] }),
  ),
}));

vi.mock('routes/utils', () => ({
  serverAddress: () => 'http://localhost',
}));

const dummy: Project = {
  archived: false,
  id: 0,
  immutable: true,
  name: '',
  notes: [],
  state: 'UNSPECIFIED',
  userId: 0,
  workspaceId: 0,
};

const setup = async () => {
  const store = new ProjectStore();
  store.fetch(1);
  await waitFor(() => {
    expect(store.getProject(1).get().getOrElse(dummy).name).toEqual('project_1');
  });
  return store;
};

describe('ProjectStore', () => {
  afterEach(() => vi.clearAllMocks());

  it('upsert', async () => {
    const store = await setup();

    store.upsert({ id: 1, name: 'project_1_new', workspace_id: 1 });
    await waitFor(() => {
      expect(store.getProject(1).get().getOrElse(dummy).name).toEqual('project_1_new');
      expect(store.getProjectsByWorkspace(1).get().getOrElse(List()).toJSON()).toEqual([
        { id: 1, name: 'project_1_new', workspaceId: 1 },
      ]);
    });

    store.upsert({ id: 1, name: 'project_1', workspace_id: 2 });
    await waitFor(() => {
      expect(store.getProject(1).get().getOrElse(dummy).workspaceId).toEqual(2);
      expect(store.getProjectsByWorkspace(1).get().getOrElse(List()).toJSON()).toHaveLength(0);
    });
  });

  it('delete', async () => {
    const store = await setup();
    store.delete(1);
    await waitFor(() => {
      expect(store.getProject(1).get()).toEqual(NotLoaded);
      expect(store.getProjectsByWorkspace(1).get().getOrElse(List()).toJSON()).toHaveLength(0);
    });
  });
});
