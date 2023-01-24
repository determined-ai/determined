import { Map } from 'immutable';
import { Observable, observable, WritableObservable } from 'micro-observables';

import { getExperiments } from 'services/api';
import { V1Pagination } from 'services/api-ts-sdk';
import { GetExperimentsParams } from 'services/types';
import { ExperimentItem } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { encodeParams } from 'utils/store';

type ExperimentPagination = {
  experiments: ExperimentItem[];
  pagination?: V1Pagination;
};

export class ExperimentsService {
  readonly #params: Readonly<GetExperimentsParams> = {};
  #experimentsCache: WritableObservable<Map<string, ExperimentPagination>> = observable(Map());

  constructor(params: GetExperimentsParams) {
    this.#params = params;
  }
  public get experiments(): Observable<Loadable<ExperimentPagination>> {
    return this.#experimentsCache.select((exp) => {
      const loadedVal = exp.get(encodeParams(this.#params));
      return loadedVal ? Loaded(loadedVal) : NotLoaded;
    });
  }

  public fetchExperiments(canceler: AbortController): () => Promise<void> {
    return async () => {
      try {
        const response = await getExperiments(this.#params, { signal: canceler.signal });
        this.#experimentsCache.update((prevState: Map<string, ExperimentPagination>) => {
          const newState = prevState.set(encodeParams(this.#params), response);
          return newState;
        });
      } catch (e) {
        handleError(e);
      }
    };
  }
}
