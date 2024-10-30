import { v4 as uuidV4 } from 'uuid';

import { safeName } from 'e2e/utils/naming';
import {
  ExperimentsApi,
  InternalApi,
  Trialv1State,
  V1ActivateExperimentResponse,
  V1ArchiveExperimentResponse,
  V1CancelExperimentResponse,
  V1CheckpointTrainingMetadata,
  V1CreateExperimentRequest,
  V1CreateExperimentResponse,
  V1CreateTrialResponse,
  V1DeleteExperimentResponse,
  V1KillExperimentResponse,
  V1PatchExperiment,
  V1PatchExperimentResponse,
  V1PatchTrialResponse,
  V1PauseExperimentResponse,
  V1PostTaskLogsResponse,
  V1ReportCheckpointResponse,
  V1ReportTrialMetricsResponse,
  V1TaskLog,
  V1TrialMetrics,
  V1UnarchiveExperimentResponse,
} from 'services/api-ts-sdk';

import { ApiArgsFixture, apiFixture } from './api';

const reportApiErrorJson = <T extends Promise<unknown>>(p: T): T => {
  p.catch(async (e) => {
    if (e instanceof Response) console.error(await e.json());
  });
  return p;
};

export class ApiRun {
  constructor(
    protected internalApi: InternalApi,
    public response: V1CreateTrialResponse,
  ) {}

  get id(): number {
    return this.response.trial.id;
  }

  get taskId(): string {
    const { taskId } = this.response.trial;
    if (typeof taskId !== 'string') {
      throw new Error('no task id found');
    }
    return taskId;
  }

  patchState(state: Trialv1State): Promise<V1PatchTrialResponse> {
    return reportApiErrorJson(this.internalApi.patchTrial(this.id, { state, trialId: this.id }));
  }

  recordLog(logs: Omit<V1TaskLog, 'taskId'>[]): Promise<V1PostTaskLogsResponse> {
    return reportApiErrorJson(
      this.internalApi.postTaskLogs({ logs: logs.map((l) => ({ ...l, taskId: this.taskId })) }),
    );
  }

  reportMetrics(
    group: 'training' | 'validation' | 'inference',
    metrics: Omit<V1TrialMetrics, 'trialId' | 'trialRunId'>,
  ): Promise<V1ReportTrialMetricsResponse> {
    return reportApiErrorJson(
      this.internalApi.reportTrialMetrics(this.id, {
        group,
        metrics: { ...metrics, trialId: this.id, trialRunId: 0 },
      }),
    );
  }

  reportCheckpoint(
    stepsCompleted: number,
    training: V1CheckpointTrainingMetadata,
    resources: { [k: string]: string } = {},
    metadata: object = {},
  ): Promise<V1ReportCheckpointResponse> {
    return reportApiErrorJson(
      this.internalApi.reportCheckpoint({
        metadata: {
          ...metadata,
          steps_completed: stepsCompleted,
        },
        resources,
        state: 'STATE_COMPLETED',
        taskId: this.taskId,
        training,
        uuid: uuidV4(),
      }),
    );
  }
}

class ApiSearch {
  runs: ApiRun[] = [];

  constructor(
    protected experimentApi: ExperimentsApi,
    protected internalApi: InternalApi,
    public response: V1CreateExperimentResponse,
  ) {}

  get id(): number {
    return this.response.experiment.id;
  }

  get externalId(): string {
    const { externalExperimentId } = this.response.experiment;
    if (typeof externalExperimentId !== 'string') {
      throw new Error('no external experiment id found');
    }
    return externalExperimentId;
  }

  patch(body: Omit<V1PatchExperiment, 'id'> = {}): Promise<V1PatchExperimentResponse> {
    return reportApiErrorJson(
      this.experimentApi.patchExperiment(this.id, { id: this.id, ...body }),
    );
  }

  /**
   * Perform an action on a search. NOTE: unmanaged experiments can only be archived, unarchived and deleted
   */
  action(action: 'pause'): Promise<V1PauseExperimentResponse>;
  action(action: 'resume' | 'activate'): Promise<V1ActivateExperimentResponse>;
  action(action: 'cancel' | 'stop'): Promise<V1CancelExperimentResponse>;
  action(action: 'kill'): Promise<V1KillExperimentResponse>;
  action(action: 'archive'): Promise<V1ArchiveExperimentResponse>;
  action(action: 'unarchive'): Promise<V1UnarchiveExperimentResponse>;
  action(action: 'delete'): Promise<V1DeleteExperimentResponse>;
  action(
    action:
      | 'activate'
      | 'archive'
      | 'unarchive'
      | 'delete'
      | 'kill'
      | 'pause'
      | 'resume'
      | 'stop'
      | 'cancel',
  ): Promise<unknown> {
    type idMethod = {
      [k in keyof ExperimentsApi]: ExperimentsApi[k] extends (id: number) => Promise<unknown>
        ? k
        : never;
    }[keyof ExperimentsApi];
    const managedMethods = ['activate', 'cancel', 'kill', 'pause', 'resume', 'stop'];
    const methodMap: Record<typeof action, idMethod> = {
      activate: 'activateExperiment',
      archive: 'archiveExperiment',
      cancel: 'cancelExperiment',
      delete: 'deleteExperiment',
      kill: 'killExperiment',
      pause: 'pauseExperiment',
      resume: 'activateExperiment',
      stop: 'cancelExperiment',
      unarchive: 'unarchiveExperiment',
    };
    if (this.response.experiment.unmanaged && managedMethods.includes(action)) {
      throw new Error(`Action ${action} only works on managed experiments`);
    }
    return reportApiErrorJson(this.experimentApi[methodMap[action]](this.id));
  }

  async addRun(hparams: unknown = { wow: 'cool' }): Promise<ApiRun> {
    const response = reportApiErrorJson(
      this.internalApi.createTrial({ experimentId: this.id, hparams, unmanaged: true }),
    );
    const apiRun = new ApiRun(this.internalApi, await response);
    this.runs.push(apiRun);
    return apiRun;
  }

  async getRuns(): Promise<ApiRun[]> {
    const trials = await this.experimentApi.getExperimentTrials(
      this.id,
      undefined,
      undefined,
      undefined,
      -1,
    );
    this.runs = trials.trials.map((trial) => new ApiRun(this.internalApi, { trial }));
    return this.runs;
  }
}

export class ApiSearchFixture extends apiFixture(InternalApi) {
  searches: ApiSearch[] = [];
  protected experimentApi: ExperimentsApi;

  constructor(
    apiArgs: ApiArgsFixture,
    public defaultProjectId: number,
  ) {
    super(apiArgs);
    this.experimentApi = new ExperimentsApi(...apiArgs);
  }

  async new(
    config: object = {},
    body: Omit<V1CreateExperimentRequest, 'config'> = {},
  ): Promise<ApiSearch> {
    const configWithDefaults = {
      entrypoint: 'echo bonjour!',
      name: safeName('apisearch'),
      searcher: {
        max_trials: 1,
        metric: 'x',
        name: 'random',
      },
      ...config,
    };
    const bodyWithDefaults = {
      activate: false,
      projectId: this.defaultProjectId,
      ...body,
    };
    const response = await reportApiErrorJson(
      this.api.createExperiment({
        ...bodyWithDefaults,
        config: JSON.stringify(configWithDefaults),
      }),
    );
    const search = new ApiSearch(this.experimentApi, this.api, response);
    this.searches.push(search);
    return search;
  }

  dispose(): Promise<V1DeleteExperimentResponse> {
    return Promise.all(this.searches.map((s) => s.action('delete')));
  }
}
