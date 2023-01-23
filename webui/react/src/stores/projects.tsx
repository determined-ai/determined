import { Map } from 'immutable';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getWorkspaceProjects } from 'services/api';
import { Project } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { observable, useObservable, WritableObservable } from 'utils/observable';

class ProjectService {
  projects: WritableObservable<Map<number, Project>> = observable(Map());
  projectsByIndex: WritableObservable<Map<string, Project[]>> = observable(Map());

  canceler: AbortController;

  constructor(canceler: AbortController) {
    this.canceler = canceler;
  }

  fetchWorkspaceProjects = async (workspaceId: number): Promise<void> => {
    try {
      const response = await getWorkspaceProjects(
        {
          id: workspaceId,
          limit: 0,
        },
        { signal: this.canceler.signal },
      );
      this.projectsByIndex.update((prevState: Map<string, Project[]>) => {
        return prevState.set(`byworkspace-${workspaceId}`, response.projects);
      });
      this.projects.update((prevState: Map<number, Project>) => {
        return prevState.withMutations((state: Map<number, Project>) => {
          response.projects.forEach((proj) => state.set(proj.id, proj));
        });
      });
    } catch (e) {
      handleError(e);
    }
  };
}

const ProjectsContext = createContext<ProjectService | null>(null);

export const ProjectsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [projectStore] = useState(() => new ProjectService(new AbortController()));

  return <ProjectsContext.Provider value={projectStore}>{children}</ProjectsContext.Provider>;
};

const useProjectsStore = (): ProjectService => {
  const store = useContext(ProjectsContext);
  if (store === null) throw new Error('useProjects is not a store');
  return store;
};

export const useFetchWorkspaceProjects = (): ((workspaceId: number) => Promise<void>) => {
  const store = useProjectsStore();

  if (store === null) {
    throw new Error('Attempted to use useWorkspaceProjects outside of Projects Context');
  }

  return useCallback(
    async (workspaceId: number): Promise<void> => {
      await store.fetchWorkspaceProjects(workspaceId);
    },
    [store],
  );
};

export const useWorkspaceProjects = (
  workspaceId: number | Loadable<number>,
): Loadable<Project[]> => {
  const store = useProjectsStore();

  if (store === null) {
    throw new Error('Attempted to use useWorkspaceProjects outside of Projects Context');
  }

  let loadedWorkspaceId: number;

  if (Loadable.isLoadable(workspaceId)) {
    if (Loadable.isLoading(workspaceId)) {
      return NotLoaded;
    } else {
      loadedWorkspaceId = workspaceId.data;
    }
  } else {
    loadedWorkspaceId = workspaceId;
  }

  const projectsByIndex = store.projectsByIndex.get();

  const projects = projectsByIndex.get(`byworkspace-${loadedWorkspaceId}`);

  if (projects === undefined) return NotLoaded;

  return Loaded(projects);
};

export const useProject = (projectId: number): Loadable<Project> => {
  const store = useProjectsStore();

  if (store === null) {
    throw new Error('Attempted to use useProject outside of Projects Context');
  }

  const projects = useObservable(store.projects);

  const project = projects.get(projectId);

  if (project === undefined) return NotLoaded;

  return Loaded(project);
};
