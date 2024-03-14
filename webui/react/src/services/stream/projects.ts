import { isEqual } from 'lodash';

import { Streamable, StreamSpec } from '.';

export class ProjectSpec extends StreamSpec {
  readonly #id: Streamable = 'projects';
  #workspace_ids: Array<number>;
  #project_ids: Array<number>;

  constructor(workspace_ids?: Array<number>, project_ids?: Array<number>) {
    super();
    this.#workspace_ids = workspace_ids?.sort() || [];
    this.#project_ids = project_ids?.sort() || [];
  }

  public equals = (sp?: StreamSpec): boolean => {
    if (!sp) return false;
    if (sp instanceof ProjectSpec) {
      return (
        isEqual(sp.#project_ids, this.#project_ids) &&
        isEqual(sp.#workspace_ids, this.#workspace_ids)
      );
    }
    return false;
  };

  public id = (): Streamable => {
    return this.#id;
  };

  public toWire = (): Record<string, Array<number>> => {
    return { project_ids: this.#project_ids, workspace_ids: this.#workspace_ids };
  };
}
