import { every, isEqual } from 'lodash';

import { Streamable, StreamSpec } from '.';

export class ExperimentSpec extends StreamSpec {
  readonly #id: Streamable = 'experiments';
  #experiment_ids: Array<number>;

  constructor(experiment_ids?: Array<number>) {
    super();
    this.#experiment_ids = experiment_ids || [];
  }

  public copy = (): ExperimentSpec => {
    return new ExperimentSpec(this.#experiment_ids);
  };

  public equals = (sp?: StreamSpec): boolean => {
    if (!sp) return false;
    if (sp instanceof ExperimentSpec) {
      return isEqual(sp.#experiment_ids, this.#experiment_ids);
    }
    return false;
  };

  public id = (): Streamable => {
    return this.#id;
  };

  public toWire = (): Record<string, Array<number>> => {
    return { experiment_ids: this.#experiment_ids };
  };

  public contains = (sp: StreamSpec): boolean => {
    if (!(sp instanceof ExperimentSpec)) return false;
    return every(sp.#experiment_ids, (i) => this.#experiment_ids.includes(i));
  };
}
