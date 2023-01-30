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

// This is singleton class
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

  // Get singleton instance
  public static getInstance(): ExperimentsService {
    if (!ExperimentsService.#instance) {
      ExperimentsService.#isInternalConstructing = true;
      ExperimentsService.#instance = new ExperimentsService();
      ExperimentsService.#isInternalConstructing = false;
    }
    return ExperimentsService.#instance;
  }

  // Get an experiment by experiment id
  public getExperimentsByIds(experimentIds: number[]): Observable<Loadable<ExperimentItem[]>> {
    return this.#experimentMap.select((map) => {
      const expList: ExperimentItem[] = experimentIds
        .map((id) => map.get(id))
        .flatMap((exp) => (exp ? [exp] : []));
      return Loaded(expList);
    });
  }

  public getExperimentsByParams(
    params: Readonly<GetExperimentsParams>,
  ): Observable<Loadable<ExperimentPagination>> {
    return this.#experimentsCache.select((map) => {
      const cache = map.get(encodeParams(params));
      if (!cache) {
        return NotLoaded;
      }
      const expMap = this.#experimentMap.get();
      const experiments: ExperimentItem[] = cache.experimentIds
        .map((id) => expMap.get(id))
        .flatMap((exp) => (exp ? [exp] : []));
      const expPagination: ExperimentPagination = {
        experiments: experiments,
        pagination: cache.pagination,
      };
      return Loaded(expPagination);
    });
  }

  // fetch experiments with params
  public fetchExperiments(
    params: Readonly<GetExperimentsParams>,
    canceler: AbortController,
  ): () => Promise<void> {
    return async () => {
      try {
        const response = await getExperiments(params, { signal: canceler.signal });
        this.#updateExperimentsCache(response, params);
        this.#updateExperimentMap(response.experiments);
      } catch (e) {
        handleError(e);
      }
    };
  }

  #updateExperimentMap(experimentItems: Readonly<ExperimentItem[]>) {
    this.#experimentMap.update((map) => {
      const newMap: Map<number, ExperimentItem> = map.withMutations((mutMap) => {
        for (const exp of experimentItems) {
          mutMap.set(exp.id, exp);
        }
      });
      return newMap;
    });
  }

  #updateExperimentsCache(
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
