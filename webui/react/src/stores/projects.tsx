import { Map } from 'immutable';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getWorkspaceProjects } from 'services/api';
import { Project } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type UpdateProjectsByIndex = (fn: (ws: Map<string, Project[]>) => Map<string, Project[]>) => void;

type ProjectsContext = {
  projects: Map<number, Project>;
  projectsByIndex: Map<string, Project[]>;
  updateProjects: (fn: (project: Map<number, Project>) => Map<number, Project>) => void;
  updateProjectsByIndex: UpdateProjectsByIndex;
};

const ProjectsContext = createContext<ProjectsContext | null>(null);

export const ProjectsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [workspaceProjects, setWorkspaceProjects] = useState<Map<string, Project[]>>(Map());
  const [projects, setProjects] = useState<Map<number, Project>>(() => Map());

  return (
    <ProjectsContext.Provider
      value={{
        projects,
        projectsByIndex: workspaceProjects,
        updateProjects: setProjects,
        updateProjectsByIndex: setWorkspaceProjects,
      }}>
      {children}
    </ProjectsContext.Provider>
  );
};

export const useFetchWorkspaceProjects = (
  canceler: AbortController,
): ((workspaceId: number) => Promise<void>) => {
  const context = useContext(ProjectsContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchWorkspaceProjects outside of Projects Context');
  }

  const { updateProjectsByIndex, updateProjects } = context;

  return useCallback(
    async (workspaceId: number): Promise<void> => {
      try {
        const response = await getWorkspaceProjects(
          {
            id: workspaceId,
            limit: 0,
          },
          { signal: canceler.signal },
        );

        updateProjectsByIndex((prevState) => {
          return prevState.set(`byworkspace-${workspaceId}`, response.projects);
        });

        updateProjects((prevState) => {
          return prevState.withMutations((state) => {
            response.projects.forEach((proj) => state.set(proj.id, proj));
          });
        });
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, updateProjectsByIndex, updateProjects],
  );
};

export const useEnsureWorkspaceProjectsFetched = (
  canceler: AbortController,
): ((workspaceId: number) => Promise<void>) => {
  const context = useContext(ProjectsContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchWorkspaceProjects outside of Projects Context');
  }

  const { updateProjectsByIndex, updateProjects } = context;

  return useCallback(
    async (workspaceId: number): Promise<void> => {
      if (context.projectsByIndex.get(`byworkspace-${workspaceId}`)) return;

      try {
        const response = await getWorkspaceProjects(
          {
            id: workspaceId,
            limit: 0,
          },
          { signal: canceler.signal },
        );

        updateProjectsByIndex((prevState) => {
          return prevState.set(`byworkspace-${workspaceId}`, response.projects);
        });

        updateProjects((prevState) => {
          return prevState.withMutations((state) => {
            response.projects.forEach((proj) => state.set(proj.id, proj));
          });
        });
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, context.projectsByIndex, updateProjectsByIndex, updateProjects],
  );
};

export const useWorkspaceProjects = (
  workspaceId: number | Loadable<number>,
): Loadable<Project[]> => {
  const context = useContext(ProjectsContext);

  if (context === null) {
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

  const { projectsByIndex } = context;

  const projects = projectsByIndex.get(`byworkspace-${loadedWorkspaceId}`);

  if (projects === undefined) return NotLoaded;

  return Loaded(projects);
};

export const useProject = (projectId: number): Loadable<Project> => {
  const context = useContext(ProjectsContext);

  if (context === null) {
    throw new Error('Attempted to use useProject outside of Projects Context');
  }

  const { projects } = context;

  const project = projects.get(projectId);

  if (project === undefined) return NotLoaded;

  return Loaded(project);
};
