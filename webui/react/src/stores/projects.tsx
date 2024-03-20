import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { List, Map } from 'immutable';

import { getWorkspaceProjects } from 'services/api';
import { Streamable, StreamContent } from 'services/stream';
import { ProjectSpec } from 'services/stream/wire';
import streamStore, { StreamSubscriber } from 'stores/stream';
import { Project } from 'types';
import asValueObject, { ValueObjectOf } from 'utils/asValueObject';
import handleError from 'utils/error';
import { immutableObservable, Observable } from 'utils/observable';

export class ProjectStore implements StreamSubscriber {
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
      const workspaceIds = this.#projectsByWorkspace
        .get()
        .keySeq()
        .map((s) => Number(s))
        .toJSON();

      streamStore.emit(new ProjectSpec([...workspaceIds, workspaceId]));
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

  public delete(id: number): void {
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

  public upsert(content: StreamContent): void {
    const p = mapStreamProject(content);
    let prevProjectWorkspaceId: number | undefined;

    this.#projects.update((prev) =>
      prev.withMutations((map) => {
        const project = map.get(p.id);
        if (project) {
          prevProjectWorkspaceId = project.workspaceId;
          this.#upsert(project, p);
          map.set(project.id, asValueObject(Project, project));
        } else {
          // TODO: We should insert the new record to store once we can stream all needed information.
          // map.set(p.id, asValueObject(Project, p));
        }
        return map;
      }),
    );
    this.#projectsByWorkspace.update((prev) =>
      prev.withMutations((map) => {
        const ws = List(map.get(p.workspaceId.toString()));
        const projectInWs = ws?.findIndex((tp) => tp.id === p.id);
        if (projectInWs >= 0) {
          // The workspaceId has not changed, just update
          const updatedProject = this.#upsert(ws.get(projectInWs)!, p);
          map.set(p.workspaceId.toString(), ws.set(projectInWs, updatedProject));
          return map;
        }
        // TODO: When workspaceId has changed, we should add it to the new workspace, since we do not have all the fields from streaming endpoint, we just fetch for the new workspace.
        if (ws) {
          this.fetch(p.workspaceId, undefined, true);
        }
        // Remove project from the old workspace
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
