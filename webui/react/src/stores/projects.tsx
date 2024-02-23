import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { List, Map } from 'immutable';

import { getWorkspaceProjects } from 'services/api';
import { Streamable, StreamContent } from 'services/stream';
import { StreamSubscriber } from 'stores/stream';
import { Project } from 'types';
import asValueObject, { ValueObjectOf } from 'utils/asValueObject';
import handleError from 'utils/error';
import { immutableObservable, Observable } from 'utils/observable';

class ProjectStore implements StreamSubscriber {
  #projects = immutableObservable<Map<number, ValueObjectOf<Project>>>(Map());
  #projectsByWorkspace = immutableObservable<Map<string, List<Project>>>(Map());

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
              response.projects.forEach((project) =>
                map.set(project.id, asValueObject(Project, project)),
              );
            }),
          );
          this.#projectsByWorkspace.update((prev) =>
            prev.set(workspaceKey, List(response.projects)),
          );
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

  public getProjectsByWorkspace(workspaceId?: number): Observable<Loadable<List<Project>>> {
    return this.#projectsByWorkspace.select((map) => {
      if (workspaceId == null) return NotLoaded;

      const projects = map.get(workspaceId.toString());
      return projects ? Loaded(projects) : NotLoaded;
    });
  }

  public delete(id: number) {
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
              const i = ws.findIndex((p) => p.id === id);
              map.set(deleted.workspaceId.toString(), ws.remove(i));
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

  public upsert(content: StreamContent) {
    const p = mapStreamProject(content);
    let prevProjectWorkspaceId: number | undefined;

    this.#projects.update((prev) =>
      prev.withMutations((map) => {
        const project = map.get(p.id);
        if (project) {
          prevProjectWorkspaceId = project.workspaceId;
          this.#upsert(project, p);
        } else {
          map.set(p.id, asValueObject(Project, p));
        }
        return map;
      }),
    );
    this.#projectsByWorkspace.update((prev) =>
      prev.withMutations((map) => {
        const ws = map.get(p.workspaceId.toString());
        const projectInWs = ws?.find((tp) => tp.id === p.id);
        if (projectInWs) {
          // The workspaceId has not changed, just update
          this.#upsert(projectInWs, p);
          return map;
        }
        // The workspaceId has changed, add to the new workspace and remove from the old workspace
        if (ws) {
          map.set(p.workspaceId.toString(), ws.push(p));
        }
        if (prevProjectWorkspaceId) {
          const ows = map.get(prevProjectWorkspaceId.toString());
          if (ows) {
            const i = ows.findIndex((op) => op.id === p.id);
            map.set(prevProjectWorkspaceId.toString(), ows.remove(i));
          }
        }
        return map;
      }),
    );
  }

  public id(): Streamable {
    return 'projects';
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
