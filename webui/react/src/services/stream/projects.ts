import { isEqual } from 'lodash';

import { Streamable } from '.';

export abstract class StreamSpec {
    abstract copy: () => StreamSpec
    abstract equals: (sp: StreamSpec) => boolean
    abstract id: () => Streamable
    abstract toWire: () => Record<string, Array<number>>
}

export class ProjectSpec extends StreamSpec {
    readonly #id: Streamable = 'projects';
    #workspace_ids: Array<number>;
    #project_ids: Array<number>;

    constructor(workspace_ids?: Array<number>, project_ids?: Array<number>) {
        super();
        this.#workspace_ids = workspace_ids || [];
        this.#project_ids = project_ids || [];
    }

    public copy = (): ProjectSpec => {
        return new ProjectSpec(this.#project_ids, this.#workspace_ids);
    };

    public equals = (sp: StreamSpec): boolean => {
        if (sp instanceof ProjectSpec) {
            return isEqual(sp.#project_ids, this.#project_ids) && isEqual(sp.#workspace_ids, this.#workspace_ids);
        }
        return false;
    };

    public id = (): Streamable => {
        return this.#id;
    };

    public toWire = (): any => {
        return { project_ids: this.#project_ids, workspace_ids: this.#workspace_ids };
    };

}
