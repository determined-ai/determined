import { Map } from 'immutable';

import { getExperiments } from 'services/api';
import { GetExperimentsParams } from 'services/types';
import { ExperimentPagination } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { Observable, observable, WritableObservable } from 'utils/observable';
import { encodeParams } from 'utils/store';

// this is singleton
export class ExperimentsService {
  static #isInternalConstructing = false;
  static #instance: ExperimentsService;
  #experimentsCache: WritableObservable<Map<string, ExperimentPagination>> = observable(Map());

  private constructor() {
    if (!ExperimentsService.#isInternalConstructing) {
      throw new TypeError('ExperimentsService is not constructable');
    }
  }

  public static getInstance(): ExperimentsService {
    if (!ExperimentsService.#instance) {
      ExperimentsService.#isInternalConstructing = true;
      ExperimentsService.#instance = new ExperimentsService();
      ExperimentsService.#isInternalConstructing = false;
    }
    return ExperimentsService.#instance;
  }

  public experimentsByParams(
    params: Readonly<GetExperimentsParams>,
  ): Observable<Loadable<ExperimentPagination>> {
    return this.#experimentsCache.select((exp) => {
      const loadedVal = exp.get(encodeParams(params));
      return loadedVal ? Loaded(loadedVal) : NotLoaded;
    });
  }

  public fetchExperiments(
    params: Readonly<GetExperimentsParams>,
    canceler: AbortController,
  ): () => Promise<void> {
    return async () => {
      try {
        const response = await getExperiments(params, { signal: canceler.signal });
        this.#experimentsCache.update((prevState: Map<string, ExperimentPagination>) => {
          const newState = prevState.set(encodeParams(params), response);
          return newState;
        });
      } catch (e) {
        handleError(e);
      }
    };
  }
}
