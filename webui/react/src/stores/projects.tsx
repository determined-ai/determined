import { Map } from 'immutable';

import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import { getWorkspaceProjects } from 'services/api';
import { Project } from 'types';
import handleError from 'utils/error';
import { Observable, observable, WritableObservable } from 'utils/observable';

class ProjectStore {
  #projects: WritableObservable<Map<number, Project>> = observable(Map());
  #projectsByWorkspace: WritableObservable<Map<string, Project[]>> = observable(Map());

  public fetch(workspaceId: number, signal?: AbortSignal, force = false): () => void {
    const workspaceKey = workspaceId.toString();
    const canceler = new AbortController();

    if (force || !this.#projectsByWorkspace.get().has(workspaceKey)) {
      getWorkspaceProjects({ id: workspaceId, limit: 0 }, { signal: signal ?? canceler.signal })
        .then((response) => {
          // Prevent unnecessary re-renders.
          if (!force && this.#projectsByWorkspace.get().has(workspaceKey)) return;
          this.#projects.update((prev) =>
            prev.withMutations((map) => {
              response.projects.forEach((project) => map.set(project.id, project));
            }),
          );
          this.#projectsByWorkspace.update((prev) => prev.set(workspaceKey, response.projects));
        })
        .catch(handleError);
    }

    return () => canceler.abort();
  }

  public getProject(projectId?: number): Observable<Loadable<Project>> {
    return this.#projects.select((map) => {
      if (projectId == null) return NotLoaded;

      const project = map.get(projectId);
      return project ? Loaded(project) : NotLoaded;
    });
  }

  public getProjectsByWorkspace(workspaceId?: number): Observable<Loadable<Project[]>> {
    return this.#projectsByWorkspace.select((map) => {
      if (workspaceId == null) return NotLoaded;

      const projects = map.get(workspaceId.toString());
      return projects ? Loaded(projects) : NotLoaded;
    });
  }
}

export default new ProjectStore();
