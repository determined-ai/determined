import { isEqual } from 'lodash';

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

  // eslint-disable-next-line  @typescript-eslint/no-explicit-any
  public toWire = (): Record<string, any> => {
    return { experiment_ids: this.#experiment_ids };
  };
}
