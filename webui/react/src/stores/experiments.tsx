import { Map } from 'immutable';

import { getExperiments } from 'services/api';
import { V1Pagination } from 'services/api-ts-sdk';
import { GetExperimentsParams } from 'services/types';
import { ExperimentItem, ExperimentPagination } from 'types';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { Observable, observable, WritableObservable } from 'utils/observable';
import { encodeParams } from 'utils/store';

import PollingStore from './polling';

type ExperimentCache = {
  experimentIds: Readonly<number[]>;
  pagination: Readonly<V1Pagination>;
};

class ExperimentStore extends PollingStore {
  // Cache values keyed by encoded request param.
  #experimentCache: WritableObservable<Map<string, ExperimentCache>> = observable(Map());
  #experimentMap: WritableObservable<Map<number, ExperimentItem>> = observable(Map());

  public getExperimentsByIds(experimentIds: number[]): Observable<Readonly<ExperimentItem[]>> {
    return this.#experimentMap.select((map) =>
      experimentIds.flatMap((id) => {
        const exp = map.get(id);
        return exp ? [exp] : [];
      }),
    );
  }

  public getExperimentsByParams(
    params: GetExperimentsParams,
  ): Observable<Loadable<Readonly<ExperimentPagination>>> {
    return Observable.select([this.#experimentCache, this.#experimentMap], (cache, map) => {
      const cachedExperiment = cache.get(encodeParams(params));
      if (!cachedExperiment) return NotLoaded;

      const experimentPagination: ExperimentPagination = {
        experiments: cachedExperiment.experimentIds.flatMap((id) => {
          const exp = map.get(id);
          return exp ? [exp] : [];
        }),
        pagination: cachedExperiment.pagination,
      };
      return Loaded(experimentPagination);
    });
  }

  protected async poll(params: GetExperimentsParams) {
    const response = await getExperiments(params, { signal: this.canceler?.signal });
    this.updateExperimentCache(response, params);
    this.updateExperimentMap(response.experiments);
  }

  private updateExperimentCache(
    pagination: Readonly<ExperimentPagination>,
    params: Readonly<GetExperimentsParams>,
  ) {
    this.#experimentCache.update((prev) =>
      prev.set(encodeParams(params), {
        experimentIds: pagination.experiments.map((exp) => exp.id),
        pagination: pagination.pagination,
      }),
    );
  }

  private updateExperimentMap(experimentItems: Readonly<ExperimentItem[]>) {
    this.#experimentMap.update((prev) =>
      prev.withMutations((map) => {
        for (const exp of experimentItems) map.set(exp.id, exp);
      }),
    );
  }
}

export default new ExperimentStore();
