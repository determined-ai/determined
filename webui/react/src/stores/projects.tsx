import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { Map } from 'immutable';
import { find, remove } from 'lodash';

import { getWorkspaceProjects } from 'services/api';
import { StreamContent } from 'services/stream';
import { ProjectSpec } from 'services/stream/projects';
import { Stream } from 'services/stream/stream';
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

  public deleteProject(id: number) {
    let deleted: Project | undefined;
    this.#projects.update((prev) =>
      prev.withMutations((map) => {
        deleted = map.get(id);
        return map.delete(id);
      }),
    );
    if (deleted) {
      this.#projectsByWorkspace.update((prev) =>
        prev.withMutations((map) => {
          if (deleted) {
            const ws = map.get(deleted.workspaceId.toString());
            if (ws) {
              remove(ws, (p) => p.id === id);
              map.set(deleted.workspaceId.toString(), [...ws]);
            }
          }
          return map;
        }),
      );
    }
  }

  #upsert(p: Project, np: Project): Project {
    p.name = np.name;
    p.description = np.description;
    p.archived = np.archived;
    p.workspaceId = np.workspaceId;
    p.state = np.state;
    return { ...p };
  }

  public upsertProject(p: Project) {
    let prevProjectWorkspaceId: number | undefined;

    this.#projects.update((prev) =>
      prev.withMutations((map) => {
        const project = map.get(p.id);
        if (project) {
          prevProjectWorkspaceId = project.workspaceId;
          this.#upsert(project, p);
        } else {
          map.set(p.id, { ...p });
        }
        return map;
      }),
    );
    this.#projectsByWorkspace.update((prev) =>
      prev.withMutations((map) => {
        const projectInWs = find(map.get(p.workspaceId.toString()), (tp) => tp.id === p.id);
        if (projectInWs) {
          // The workspaceId has not changed, just update
          this.#upsert(projectInWs, p);
          return map;
        }
        // The workspaceId has changed, add to the new workspace and remove from the old workspace
        const ws = map.get(p.workspaceId.toString());
        if (ws) {
          ws.push(p);
          map.set(p.workspaceId.toString(), [...ws]);
        }
        if (prevProjectWorkspaceId) {
          const ows = map.get(prevProjectWorkspaceId.toString());
          if (ows) {
            remove(ows, (op) => op.id === p.id);
            map.set(prevProjectWorkspaceId.toString(), [...ows]);
          }
        }
        return map;
      }),
    );
  }

  public subscribe(stream: Stream, spec: ProjectSpec) {
    stream.subscribe(spec);
  }
}

export default new ProjectStore();

export const mapStreamProject = (p: StreamContent): Project => ({
  archived: p.archived,
  description: p.description,
  id: p.id,
  immutable: p.immutable,
  name: p.name,
  notes: p.notes,
  state: p.state,
  userId: p.user_id,
  workspaceId: p.workspace_id,
});
