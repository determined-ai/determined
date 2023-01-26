import { Map } from 'immutable';

import { getExperiments } from 'services/api';
import { V1Pagination } from 'services/api-ts-sdk';
import { GetExperimentsParams } from 'services/types';
import { ExperimentItem, ExperimentPagination } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { Observable, observable, WritableObservable } from 'utils/observable';
import { encodeParams } from 'utils/store';

type ExperimentsCache = {
  experimentIds: Readonly<number[]>;
  pagination: Readonly<V1Pagination>;
};

// this is singleton
export class ExperimentsService {
  static #isInternalConstructing = false;
  static #instance: ExperimentsService;
  #experimentsCache: WritableObservable<Map<string, ExperimentsCache>> = observable(Map());
  #experimentMap: WritableObservable<Map<number, ExperimentItem>> = observable(Map());

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
    return this.#experimentsCache.select((map) => {
      const cache = map.get(encodeParams(params));
      if (!cache) {
        return NotLoaded;
      }
      const experiments: ExperimentItem[] = [];
      const expMap = this.#experimentMap.get();
      for (const id of cache.experimentIds) {
        const expItem = expMap.get(id);
        if (expItem) {
          experiments.push(expItem);
        }
      }
      const expPagination: ExperimentPagination = {
        experiments: experiments,
        pagination: cache.pagination,
      };
      return Loaded(expPagination);
    });
  }

  public fetchExperiments(
    params: Readonly<GetExperimentsParams>,
    canceler: AbortController,
  ): () => Promise<void> {
    return async () => {
      try {
        const response = await getExperiments(params, { signal: canceler.signal });
        this.#updateexperimentsCache(response, params);
        this.#updateExperimentMap(response);
      } catch (e) {
        handleError(e);
      }
    };
  }

  #updateExperimentMap(expPagination: Readonly<ExperimentPagination>) {
    this.#experimentMap.update((map) => {
      let newMap: Map<number, ExperimentItem> = Map<number, ExperimentItem>();
      for (const exp of expPagination.experiments) {
        newMap = map.set(exp.id, exp);
      }
      return newMap;
    });
  }

  #updateexperimentsCache(
    expPagination: Readonly<ExperimentPagination>,
    params: Readonly<GetExperimentsParams>,
  ) {
    this.#experimentsCache.update((prevState: Map<string, ExperimentsCache>) => {
      const experimentIds: Readonly<number[]> = expPagination.experiments.map((exp) => exp.id);
      const expCache: ExperimentsCache = { experimentIds, pagination: expPagination.pagination };
      const newState = prevState.set(encodeParams(params), expCache);
      return newState;
    });
  }
}
