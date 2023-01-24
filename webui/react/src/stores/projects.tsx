import { Map } from 'immutable';
import React, {
  createContext,
  PropsWithChildren,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react';

import { getWorkspaceProjects } from 'services/api';
import { Project } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { observable, useObservable, WritableObservable } from 'utils/observable';

class ProjectService {
  private _projects: WritableObservable<Map<number, Project>> = observable(Map());
  private _projectsByIndex: WritableObservable<Map<string, Project[]>> = observable(Map());

  genWorkspaceKey = (workspaceId: number): string => `byworkspace-${workspaceId}`;

  fetchWorkspaceProjects = async (
    workspaceId: number,
    canceler: AbortController,
    forceFetch = false,
  ): Promise<void> => {
    if (!forceFetch && this._projectsByIndex.get().get(this.genWorkspaceKey(workspaceId))) return;
    try {
      const response = await getWorkspaceProjects(
        {
          id: workspaceId,
          limit: 0,
        },
        { signal: canceler.signal },
      );
      // Prevent unecessary re-renders
      if (this._projectsByIndex.get().get(this.genWorkspaceKey(workspaceId))) return;
      this._projectsByIndex.update((prevState: Map<string, Project[]>) => {
        return prevState.set(this.genWorkspaceKey(workspaceId), response.projects);
      });
      this._projects.update((prevState: Map<number, Project>) => {
        return prevState.withMutations((state: Map<number, Project>) => {
          response.projects.forEach((proj) => state.set(proj.id, proj));
        });
      });
    } catch (e) {
      handleError(e);
    }
  };

  getWorkspaceProject = (workspaceId: number): Project[] | undefined => {
    return useObservable(this._projectsByIndex).get(this.genWorkspaceKey(workspaceId));
  };

  getProject = (projectId: number): Project | undefined => {
    return useObservable(this._projects).get(projectId);
  };
}

const ProjectsContext = createContext<ProjectService | null>(null);

export const ProjectsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [store] = useState(() => new ProjectService());

  return <ProjectsContext.Provider value={store}>{children}</ProjectsContext.Provider>;
};

const useProjectsStore = (): ProjectService => {
  const store = useContext(ProjectsContext);
  if (store === null) throw new Error('useProjects is not a store');
  return store;
};

export const useFetchWorkspaceProjects = (
  canceler: AbortController,
): ((workspaceId: number) => Promise<void>) => {
  const store = useProjectsStore();

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  if (store === null) {
    throw new Error('Attempted to use useFetchWorkspaceProjects outside of Projects Context');
  }

  return useCallback(
    async (workspaceId: number): Promise<void> => {
      await store.fetchWorkspaceProjects(workspaceId, canceler, true);
    },
    [store, canceler],
  );
};

export const useEnsureWorkspaceProjectsFetched = (
  canceler: AbortController,
): ((workspaceId: number) => Promise<void>) => {
  const store = useProjectsStore();

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  if (store === null) {
    throw new Error(
      'Attempted to use useEnsureWorkspaceProjectsFetched outside of Projects Context',
    );
  }

  return useCallback(
    async (workspaceId: number): Promise<void> => {
      await store.fetchWorkspaceProjects(workspaceId, canceler);
    },
    [store, canceler],
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

  const projects = store.getWorkspaceProject(loadedWorkspaceId);

  if (projects === undefined) return NotLoaded;

  return Loaded(projects);
};

export const useProject = (projectId: number): Loadable<Project> => {
  const store = useProjectsStore();

  if (store === null) {
    throw new Error('Attempted to use useProject outside of Projects Context');
  }

  const project = store.getProject(projectId);

  if (project === undefined) return NotLoaded;

  return Loaded(project);
};
