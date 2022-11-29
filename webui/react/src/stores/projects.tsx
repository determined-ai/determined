import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getWorkspaceProjects } from 'services/api';
import { Project } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type UpdateProjects = (fn: (project: Map<number, Project>) => Map<number, Project>) => void;

type ProjectsContext = {
  projects: Map<number, Project>;
  updateProjects: (fn: (project: Map<number, Project>) => Map<number, Project>) => void;
  updateWorkspaceProjects: (fn: (ws: Loadable<Map<number, Project[]>>) => Loadable<Map<number, Project[]>>) => void;
  workspaceProjects: Loadable<Map<number, Project[]>>;
};

const ProjectsContext = createContext<ProjectsContext | null>(null);

export const ProjectsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [workspaceProjects, setWorkspaceProjects] = useState<Loadable<Map<number, Project[]>>>(NotLoaded);
  const [projects, setProjects] = useState<Map<number, Project>>(() => new Map());

  return (
    <ProjectsContext.Provider value={{
      projects,
      updateProjects: setProjects,
      updateWorkspaceProjects: setWorkspaceProjects,
      workspaceProjects: workspaceProjects,
    }}>
      {children}
    </ProjectsContext.Provider>
  );
};

const mapWorkspaceProjects = (wsProjects: Project[], updateProjects: UpdateProjects) => {
  const projectsMap = new Map<number, Project>();

  wsProjects.forEach((project) => projectsMap.set(project.id, project));

  updateProjects(() => projectsMap);
};

export const useFetchWorkspaceProjects = (canceler: AbortController): ((workspaceId: number) => Promise<void>) => {
  const context = useContext(ProjectsContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchProjects outside of Projects Context');
  }

  const { updateWorkspaceProjects, updateProjects } = context;

  return useCallback(async (workspaceId: number): Promise<void> => {
    try {
      const response = await getWorkspaceProjects({
        id: workspaceId,
        limit: 0,
      }, { signal: canceler.signal });

      updateWorkspaceProjects(() => {
        const projectsMap = new Map<number, Project[]>();

        projectsMap.set(workspaceId, response.projects);

        return Loaded(projectsMap);
      });

      mapWorkspaceProjects(response.projects, updateProjects);
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateWorkspaceProjects, updateProjects]);
};

export const useEnsureWorkspaceProjectsFetched = (canceler: AbortController): ((workspaceId: number) => Promise<void>) => {
  const context = useContext(ProjectsContext);

  if (context === null) {
    throw new Error('Attempted to use useEnsureFetchWorkspaces outside of Workspace Context');
  }

  const { workspaceProjects, updateWorkspaceProjects, updateProjects } = context;

  return useCallback(async (workspaceId: number): Promise<void> => {
    const projectsMap = Loadable.getOrElse(new Map<number, Project[]>(), workspaceProjects);

    if (workspaceProjects !== NotLoaded && !!projectsMap.get(workspaceId)) return;

    try {
      const response = await getWorkspaceProjects({
        id: workspaceId,
        limit: 0,
      }, { signal: canceler.signal });

      projectsMap.set(workspaceId, response.projects);
      const projectsCollection = Array.from(projectsMap.values()).flat();

      updateWorkspaceProjects(() => Loaded(projectsMap));
      mapWorkspaceProjects(projectsCollection, updateProjects);
    } catch (e) {
      handleError(e);
    }
  }, [canceler, workspaceProjects, updateWorkspaceProjects, updateProjects]);
};

export const useWorkspaceProjects = (): Map<number, Project[]> | null => {
  const context = useContext(ProjectsContext);

  if (context === null) {
    throw new Error('Attempted to use useWorkspaceProjects outside of Projects Context');
  }

  const { workspaceProjects } = context;

  const wsProjectsState = Loadable.map(workspaceProjects, (wsPj) => wsPj);

  if (wsProjectsState === NotLoaded) return null;

  const projectsMap = Loadable.getOrElse(new Map<number, Project[]>(), workspaceProjects);

  return projectsMap;
};

export const useProjects = (projectId: number): Project | undefined => {
  const context = useContext(ProjectsContext);

  if (context === null) {
    throw new Error('Attempted to use useProject outside of Projects Context');
  }

  const { projects } = context;

  const project = projects.get(projectId);

  return project;
};
