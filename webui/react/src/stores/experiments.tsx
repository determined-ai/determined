import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { Map } from 'immutable';
import * as t from 'io-ts';

import { getExperiments } from 'services/api';
import { GetExperimentsParams } from 'services/types';
import { ExperimentItem, ExperimentPagination } from 'types';
import asValueObject, { ValueObjectOf } from 'utils/asValueObject';
import { immutableObservable, Observable } from 'utils/observable';
import { encodeParams } from 'utils/store';

import PollingStore from './polling';

const experimentCacheCodec = t.type({
  experimentIds: t.readonlyArray(t.number),
  pagination: t.readonly(
    t.partial({
      endIndex: t.number,
      limit: t.number,
      offset: t.number,
      startIndex: t.number,
      total: t.number,
    }),
  ),
});
type ExperimentCache = t.TypeOf<typeof experimentCacheCodec>;

class ExperimentStore extends PollingStore {
  // Cache values keyed by encoded request param.
  #experimentCache = immutableObservable<Map<string, ValueObjectOf<ExperimentCache>>>(Map());
  #experimentMap = immutableObservable<Map<number, ValueObjectOf<ExperimentItem>>>(Map());

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
      prev.set(
        encodeParams(params),
        asValueObject(experimentCacheCodec, {
          experimentIds: pagination.experiments.map((exp) => exp.id),
          pagination: pagination.pagination,
        }),
      ),
    );
  }

  private updateExperimentMap(experimentItems: Readonly<ExperimentItem[]>) {
    this.#experimentMap.update((prev) =>
      prev.withMutations((map) => {
        for (const exp of experimentItems) map.set(exp.id, asValueObject(ExperimentItem, exp));
      }),
    );
  }
}

export default new ExperimentStore();
