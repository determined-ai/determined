import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { Map } from 'immutable';
import * as t from 'io-ts';

import { valueof } from 'ioTypes';
import { getExperiments } from 'services/api';
import { Jobv1State } from 'services/api-ts-sdk';
import { GetExperimentsParams } from 'services/types';
import {
  CheckpointStorageType,
  ExperimentItem,
  ExperimentPagination,
  ExperimentSearcherName,
  HyperparameterBase,
  Hyperparameters,
  HyperparameterType,
  JsonObject,
  RunState,
} from 'types';
import asValueObject, { ValueObjectOf } from 'utils/asValueObject';
import { immutableObservable, Observable } from 'utils/observable';
import { encodeParams } from 'utils/store';

import PollingStore from './polling';

const primitivesCodec = t.union([t.boolean, t.number, t.string]);
const searcherCodec = t.intersection([
  t.partial({
    max_length: t.record(
      t.union([t.literal('batches'), t.literal('records'), t.literal('epochs')]),
      t.number,
    ),
    max_trials: t.number,
    sourceTrialId: t.number,
  }),
  t.type({
    metric: t.string,
    name: valueof(ExperimentSearcherName),
    smallerIsBetter: t.boolean,
  }),
]);
const hyperparametersTypeCodec = valueof(HyperparameterType);
const hyperparametersCodec: t.Type<Hyperparameters> = t.recursion('Hyperparameters', () =>
  t.record(t.string, t.union([hyperparametersCodec, hyperparameterBaseCodec])),
);
const hyperparameterBaseBaseCodec = t.recursion<HyperparameterBase>('HyperparametersBase', () =>
  t.partial({
    base: t.number,
    count: t.number,
    maxval: t.number,
    minval: t.number,
    vals: t.array(primitivesCodec),
  }),
);
const hyperparameterBaseCodec = t.intersection([
  hyperparameterBaseBaseCodec,
  t.partial({
    type: hyperparametersTypeCodec,
    val: t.union([primitivesCodec, hyperparametersCodec]),
  }),
]);
const hyperparameterCodec = t.intersection([
  hyperparameterBaseBaseCodec,
  t.partial({
    val: primitivesCodec,
  }),
  t.type({
    type: hyperparametersTypeCodec,
  }),
]);
const checkpointStorageCodec = t.intersection([
  t.partial({
    bucket: t.string,
    hostPath: t.string,
    storagePath: t.string,
    type: valueof(CheckpointStorageType),
  }),
  t.type({
    saveExperimentBest: t.number,
    saveTrialBest: t.number,
    saveTrialLatest: t.number,
  }),
]);
const experimentConfigCodec = t.intersection([
  t.partial({
    checkpointStorage: checkpointStorageCodec,
    description: t.string,
    labels: t.array(t.string),
    profiling: t.type({
      enabled: t.boolean,
    }),
  }),
  t.type({
    checkpointPolicy: t.string,
    hyperparameters: hyperparametersCodec,
    maxRestarts: t.number,
    name: t.string,
    resources: t.partial({
      maxSlots: t.number,
    }),
    searcher: searcherCodec,
  }),
]);
const jobSummaryCodec = t.type({
  jobsAhead: t.number,
  state: valueof(Jobv1State),
});
const experimentItemCodec = t.intersection([
  t.partial({
    checkpoints: t.number,
    checkpointSize: t.number,
    description: t.string,
    duration: t.number,
    endTime: t.string,
    externalExperimentId: t.string,
    externalTrialId: t.string,
    forkedFrom: t.number,
    jobSummary: jobSummaryCodec,
    modelDefinitionSize: t.number,
    notes: t.string,
    progress: t.number,
    projectName: t.string,
    searcherMetricValue: t.number,
    trialIds: t.array(t.number),
    unmanaged: t.boolean,
    workspaceName: t.string,
  }),
  t.type({
    archived: t.boolean,
    config: experimentConfigCodec,
    configRaw: JsonObject,
    hyperparameters: t.record(t.string, hyperparameterCodec),
    id: t.number,
    jobId: t.string,
    labels: t.array(t.string),
    name: t.string,
    numTrials: t.number,
    projectId: t.number,
    resourcePool: t.string,
    searcherType: t.string,
    startTime: t.string,
    state: t.union([valueof(RunState), valueof(Jobv1State)]),
    userId: t.number,
  }),
]);

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
        for (const exp of experimentItems) map.set(exp.id, asValueObject(experimentItemCodec, exp));
      }),
    );
  }
}

export default new ExperimentStore();
