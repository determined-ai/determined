import { Observable, observable, WritableObservable } from 'micro-observables';

import {
  archiveWorkspace,
  createWorkspace,
  deleteWorkspace,
  getWorkspaces,
  pinWorkspace,
  unarchiveWorkspace,
  unpinWorkspace,
} from 'services/api';
import { V1PostWorkspaceRequest } from 'services/api-ts-sdk';
import { GetWorkspacesParams } from 'services/types';
import { Workspace } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import PollingStore from './polling';

class WorkspaceStore extends PollingStore {
  #loadableWorkspaces: WritableObservable<Loadable<Workspace[]>> = observable(NotLoaded);

  public readonly workspaces = this.#loadableWorkspaces.readOnly();

  public readonly unarchived = this.#loadableWorkspaces.select((loadable) => {
    return Loadable.quickMatch(loadable, NotLoaded, (workspaces) => {
      return Loaded(workspaces.filter((workspace) => !workspace.archived));
    });
  });

  public readonly pinned = this.#loadableWorkspaces.select((loadable) => {
    return Loadable.quickMatch(loadable, NotLoaded, (workspaces) => {
      return Loaded(workspaces.filter((workspace) => workspace.pinned));
    });
  });

  public getWorkspace(id?: number): Observable<Loadable<Workspace>> {
    return this.workspaces.select((loadable) => {
      return Loadable.quickMatch(loadable, NotLoaded, (workspaces) => {
        const workspace = workspaces.find((workspace) => workspace.id === id);
        return workspace ? Loaded(workspace) : NotLoaded;
      });
    });
  }

  public archiveWorkspace(id: number): Promise<void> {
    return archiveWorkspace({ id }).then(() =>
      this.#loadableWorkspaces.update((loadable) =>
        Loadable.map(loadable, (workspaces) => {
          return workspaces.map((workspace) => {
            return workspace.id === id ? { ...workspace, archived: true } : workspace;
          });
        }),
      ),
    );
  }

  public unarchiveWorkspace(id: number): Promise<void> {
    return unarchiveWorkspace({ id }).then(() =>
      this.#loadableWorkspaces.update((loadable) =>
        Loadable.map(loadable, (workspaces) => {
          return workspaces.map((workspace) => {
            return workspace.id === id ? { ...workspace, archived: false } : workspace;
          });
        }),
      ),
    );
  }

  public pinWorkspace(id: number): Promise<void> {
    return pinWorkspace({ id }).then(() =>
      this.#loadableWorkspaces.update((loadable) =>
        Loadable.map(loadable, (workspaces) => {
          return workspaces.map((workspace) => {
            return workspace.id === id
              ? { ...workspace, pinned: true, pinnedAt: new Date() }
              : workspace;
          });
        }),
      ),
    );
  }

  public unpinWorkspace(id: number): Promise<void> {
    return unpinWorkspace({ id }).then(() =>
      this.#loadableWorkspaces.update((loadable) =>
        Loadable.map(loadable, (workspaces) => {
          return workspaces.map((workspace) => {
            return workspace.id === id ? { ...workspace, pinned: false } : workspace;
          });
        }),
      ),
    );
  }

  public createWorkspace(params: V1PostWorkspaceRequest): Promise<Workspace> {
    return createWorkspace(params).then((workspace) => {
      this.#loadableWorkspaces.update((loadable) => {
        return Loadable.map(loadable, (workspaces) => [...workspaces, workspace]);
      });
      return workspace;
    });
  }

  public deleteWorkspace(id: number): Promise<void> {
    return deleteWorkspace({ id }).then(() =>
      this.#loadableWorkspaces.update((loadable) =>
        Loadable.map(loadable, (workspaces) => {
          return workspaces.filter((workspace) => workspace.id !== id);
        }),
      ),
    );
  }

  public fetch(settings = {} as GetWorkspacesParams, signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getWorkspaces(settings, { signal: signal ?? canceler.signal })
      .then((response) => this.#loadableWorkspaces.set(Loaded(response.workspaces)))
      .catch(handleError);

    return () => canceler.abort();
  }

  public reset() {
    this.#loadableWorkspaces.set(NotLoaded);
  }

  protected async poll(settings: GetWorkspacesParams = {}) {
    const response = await getWorkspaces(settings, { signal: this.canceler?.signal });
    this.#loadableWorkspaces.set(Loaded(response.workspaces));
  }
}

export default new WorkspaceStore();
