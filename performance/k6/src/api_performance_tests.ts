import { sleep } from "k6";
import { Options, Scenario, Threshold } from "k6/options";
import { jUnit, textSummary } from "./utils/k6-summary";
import {
  authenticateVU,
  generateSlackResults,
  test,
  testRequestWithSession,
} from "./utils/helpers";
import {
  Results,
  TestConfiguration,
  SeededData,
  Authorization,
  TestGroup,
} from "./utils/types";

const DEFAULT_CLUSTER_URL = "http://localhost:8080";

// Fallback to localhost if a cluster url is not supplied
const CLUSTER_URL = __ENV.DET_MASTER ?? DEFAULT_CLUSTER_URL;

const RBAC_ENABLED = false;

export const setup = (skipAuth: boolean = false): TestConfiguration => {
  const resourcePool = __ENV.resource_pool;

  const model = {
    name: __ENV.model_name,
    versionNum: __ENV.model_version_number,
  };

  const trial = {
    id: __ENV.trial_id,
  };

  const workspace = {
    id: __ENV.workspace_id ?? "1",
    projectId: __ENV.project_id ?? "1",
  };

  const experiment = {
    id: __ENV.experiment_id,
    metricName: __ENV.metric_name,
    metricType: __ENV.metric_type,
    batches: __ENV.batches,
    batchesMargin: __ENV.batches_margin,
  };

  const task = {
    id: __ENV.task_id,
  };

  const seededData: SeededData = {
    model,
    trial,
    experiment,
    workspace,
    task,
    resourcePool,
  };

  const auth: Authorization = { token: skipAuth ? "" : authenticateVU(CLUSTER_URL) };
  const testConfig: TestConfiguration = { auth, seededData };
  getloadTests(testConfig, true);
  return testConfig;
};

// List of tests
const getloadTests = (
  testConfig?: TestConfiguration,
  inSetupPhase: boolean = false,
): TestGroup[] => {
  const sD = testConfig?.seededData;
  const testRequest = testRequestWithSession(CLUSTER_URL, testConfig);
  const getRequest = testRequest.bind(this, 'GET');
  const testSuite = [
    // Query the master endpoint
    test("get master configuration", getRequest("/api/v1/master")),
    test("get agents", getRequest("/api/v1/agents")),
    test("get workspaces", getRequest("/api/v1/workspaces")),
    test("get user settings", getRequest("/api/v1/users/setting")),
    test("get resource pools", getRequest("/api/v1/resource-pools")),
    test(
      "get available workspace resource pools",
      getRequest(
        `/api/v1/workspaces/${sD?.workspace.id}/available-resource-pools`,
      ),

      !!sD?.workspace.id,
    ),
    test("get users", getRequest("/api/v1/users")),
    test(
      "get workspace bindings",
      getRequest(
        `/api/v1/resource-pools/${sD?.resourcePool}/workspace-bindings`,
      ),
      !!sD?.resourcePool,
    ),
    test("login", testRequest(
      "POST",
      "/api/v1/auth/login",
      JSON.stringify({
        username: __ENV.DET_ADMIN_USERNAME,
        password: __ENV.DET_ADMIN_PASSWORD,
      }),
      {
        headers: { "Authorization": "" },
      },
    )),
    test(
      "get workspace model labels",
      getRequest(`/api/v1/model/labels?workspaceId=${sD?.workspace.id}`),
      !!sD?.workspace.id,
    ),
    test("get models", getRequest("/api/v1/models")),
    test("get telemetry", getRequest("/api/v1/master/telemetry")),
    test("get tensorboards", getRequest("/api/v1/tensorboards")),
    test("get shells", getRequest("/api/v1/shells")),
    test("get notebooks", getRequest("/api/v1/notebooks")),
    test("get commands", getRequest("/api/v1/commands")),
    test("get job queue stats", getRequest("/api/v1/job-queues/stats")),
    test(
      "get v2 job queue",
      getRequest(`/api/v1/job-queues-v2?resourcePool=${sD?.resourcePool}`),
      !!sD?.resourcePool,
    ),
    test(
      "get project",
      getRequest(`/api/v1/projects/${sD?.workspace.projectId}`),
      !!sD?.workspace.projectId,
    ),
    test("get user activity", getRequest("/api/v1/user/projects/activity")),
    test(
      "get workspace tensorboards",
      getRequest(
        `/api/v1/tensorboards?sortBy=SORT_BY_UNSPECIFIED&orderBy=ORDER_BY_UNSPECIFIED&workspaceId=${sD?.workspace.id}`,
      ),
      !!sD?.workspace.id,
    ),
    test(
      "get workspace shells",
      getRequest(
        `/api/v1/shells?sortBy=SORT_BY_UNSPECIFIED&orderBy=ORDER_BY_UNSPECIFIED&workspaceId=${sD?.workspace.id}`,
      ),
      !!sD?.workspace.id,
    ),
    test(
      "get workspace notebooks",
      getRequest(
        `/api/v1/notebooks?limit=1000&workspaceId=${sD?.workspace.id}`,
      ),
      !!sD?.workspace.id,
    ),
    test(
      "get workspace commands",
      getRequest(`/api/v1/shells?limit=1000&workspaceId=${sD?.workspace.id}`),
      !!sD?.workspace.id,
    ),
    test(
      "get workspace projects",
      getRequest(`/api/v1/workspaces/${sD?.workspace.id}/projects`),
      !!sD?.workspace.id,
    ),
    test("get webhooks", getRequest("/api/v1/webhooks")),
    test(
      "get project metric ranges",
      getRequest(
        `/api/v1/projects/${sD?.workspace.projectId}/experiments/metric-ranges`,
      ),
      !!sD?.workspace.projectId,
    ),
    test(
      "get project columns for runs table",
      getRequest(
        `/api/v1/projects/${sD?.workspace.projectId}/columns?table_type=TABLE_TYPE_RUN`,
      ),
      !!sD?.workspace.projectId,
    ),
    // This is a bad endpoint and we know it is bad.
    // No sense in making other endpoints be slowed before we get the fix in
    // https://hpe-aiatscale.atlassian.net/browse/DET-10114
    // https://hpe-aiatscale.atlassian.net/browse/DET-10115
    //test(
    //  "get project columns",
    //  getRequest(`/api/v1/projects/${sD?.workspace.projectId}/columns`),
    //),
    test(
      "search experiments",
      getRequest(
        `/api/v1/experiments-search?projectId=${sD?.workspace.projectId}`,
      ),
      !!sD?.workspace.projectId,
    ),
    test("get model labels", getRequest("/api/v1/model/labels")),
    test(
      "get model versions",
      getRequest(`/api/v1/models/${sD?.model.name}/versions`),
      !!sD?.model.name,
    ),
    test(
      "get model version",
      getRequest(
        `/api/v1/models/${sD?.model.name}/versions/${sD?.model.versionNum}`,
      ),
      !!sD?.model.versionNum && !!sD?.model.name,
    ),
    test(
      "get trial",
      getRequest(`/api/v1/trials/${sD?.trial.id}`),
      !!sD?.trial.id,
    ),
    test(
      "get trial workloads",
      getRequest(`/api/v1/trials/${sD?.trial.id}/workloads`),
      !!sD?.trial.id,
    ),
    test(
      "get trial logs",
      getRequest(`/api/v1/trials/${sD?.trial.id}/logs/fields?`),
      !!sD?.trial.id,
    ),
    test(
      "get experiment",
      getRequest(`/api/v1/experiments/${sD?.experiment.id}`),
      !!sD?.experiment.id,
    ),
    test(
      "get experiment metric names",
      getRequest(
        `/api/v1/experiments/metrics-stream/metric-names?ids=${sD?.experiment.id}`,
      ),
      !!sD?.experiment.id,
    ),
    // These endpoints will never complete on an experiment with a lot of metrics.
    // TODO fix this.
    //test(
    //  "get experiment metric batches",
    //  getRequest(
    //    `/api/v1/experiments/${sD?.experiment.id}/metrics-stream/batches?metricName=${sD?.experiment.metricName}&metricType=${sD?.experiment.metricType}`,
    //  ),
    //  !!sD?.experiment.id &&
    //    !!sD?.experiment.metricName &&
    //    !!sD?.experiment.metricType,
    //),
    // test(
    //   "get experiment trials sample",
    //   getRequest(
    //     `/api/v1/experiments/${sD?.experiment.id}/metrics-stream/trials-sample?metricName=${sD?.experiment.metricName}&metricType=${sD?.experiment.metricType}`,
    //   ),
    //   !!sD?.experiment.id &&
    //     !!sD?.experiment.metricName &&
    //     !!sD?.experiment.metricType,
    // ),
    // test(
    //   "get experiment trials snapshot",
    //   getRequest(
    //     `/api/v1/experiments/${sD?.experiment.id}/metrics-stream/trials-snapshot?metricName=${sD?.experiment.metricName}&metricType=${sD?.experiment.metricType}&batchesProcessed=${sD?.experiment.batches}&batchesMargin=${sD?.experiment.batchesMargin}`,
    //   ),
    //   !!sD?.experiment.id &&
    //     !!sD?.experiment.metricName &&
    //     !!sD?.experiment.metricType &&
    //     !!sD?.experiment.batches &&
    //     !!sD?.experiment.batchesMargin,
    // ),
    test(
      "get experiment trials",
      getRequest(`/api/v1/experiments/${sD?.experiment.id}/trials`),
      !!sD?.experiment.id,
    ),
    test(
      "get trials time series",
      getRequest(
        `/api/v1/trials/time-series?trialIds=${sD?.trial.id}&startBatches=0&metricType=METRIC_TYPE_UNSPECIFIED`,
      ),
      !!sD?.trial.id,
    ),
    test(
      "get experiment file tree",
      getRequest(`/api/v1/experiments/${sD?.experiment.id}/file_tree`),
      !!sD?.experiment.id,
    ),
    test(
      "get experiments",
      getRequest(`/api/v1/experiments?showTrialData=true`),
    ),
    test(
      "get master logs",
      getRequest(`/api/v1/master/logs?offset=-1&limit=0`),
    ),
    test(
      "get resource allocations",
      getRequest(
        `/api/v1/resources/allocation/aggregated?startDate=2000-01-01&endDate=${new Date()
          .toJSON()
          .slice(0, 10)}&period=RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY`,
      ),
    ),
    test("get tasks", getRequest(`/api/v1/tasks`)),
    test("get task count", getRequest(`/api/v1/tasks/count`)),
    test("get task", getRequest(`/api/v1/tasks/${sD?.task.id}`), !!sD?.task.id),
    test(
      "get task log fields",
      getRequest(`/api/v1/tasks/${sD?.task.id}/logs/fields`),
      !!sD?.task.id,
    ),
    test(
      "get task logs",
      getRequest(`/api/v1/tasks/${sD?.task.id}/logs`),
      !!sD?.task.id,
    ),
    test(
      "get permissions summary",
      getRequest("/api/v1/permissions-summary"),
      RBAC_ENABLED,
    ),
    test("search groups", getRequest("/api/v1/groups/search"), RBAC_ENABLED),
    test(
      "get workspace roles",
      getRequest(`/api/v1/roles/workspace/${sD?.workspace.id}`),
      RBAC_ENABLED && !!sD?.workspace.id,
    ),
    test(
      "get roles by assignability",
      getRequest("/api/v1/roles/search/by-assignability"),
      RBAC_ENABLED,
    ),
    test("get group", getRequest(`/api/v1/groups/{groupId}`), RBAC_ENABLED),
    test("search groups", getRequest("/api/v1/groups/search"), RBAC_ENABLED),
    test("search roles", getRequest(`/api/v1/roles-search`), RBAC_ENABLED),
    test("compare runs", getRequest(`/api/v1/trials/time-series?&trialIds=216
    &trialIds=218
    &trialIds=219
    &trialIds=220
    &trialIds=221
    &trialIds=223
    &trialIds=225
    &trialIds=226
    &trialIds=227
    &trialIds=228
    &trialIds=229
    &trialIds=230
    &trialIds=231
    &trialIds=232
    &trialIds=233
    &trialIds=234
    &trialIds=235
    &trialIds=236
    &trialIds=239
    &trialIds=240
    &trialIds=242
    &trialIds=246
    &trialIds=247
    &trialIds=249
    &trialIds=275
    &trialIds=276
    &trialIds=277
    &trialIds=278
    &trialIds=279
    &trialIds=280
    &trialIds=281
    &trialIds=282
    &trialIds=283
    &trialIds=284
    &trialIds=285
    &trialIds=286
    &trialIds=287
    &trialIds=288
    &trialIds=289
    &trialIds=290
    &trialIds=291
    &trialIds=292
    &trialIds=293
    &trialIds=294
    &trialIds=295
    &trialIds=296
    &trialIds=297
    &trialIds=298
    &trialIds=299
    &trialIds=300
    &trialIds=301
    &trialIds=302
    &trialIds=303
    &trialIds=304
    &trialIds=305
    &trialIds=306
    &trialIds=307
    &trialIds=308
    &trialIds=309
    &trialIds=310
    &trialIds=311
    &trialIds=312
    &trialIds=313
    &trialIds=314
    &trialIds=315
    &trialIds=316
    &trialIds=317
    &trialIds=318
    &trialIds=319
    &trialIds=320
    &trialIds=321
    &trialIds=322
    &trialIds=323
    &trialIds=324
    &trialIds=325
    &trialIds=326
    &trialIds=327
    &trialIds=328
    &trialIds=329
    &trialIds=330&maxDatapoints=1500&startBatches=0&metricType=METRIC_TYPE_UNSPECIFIED&metricIds=training.categorical_accuracy&metricIds=training.loss&metricIds=validation.val_categorical_accuracy&metricIds=validation.val_loss`), RBAC_ENABLED),
    test(
      "search roles by group",
      getRequest(`/api/v1/roles/search/by-group/{groupId}`),
      RBAC_ENABLED,
    ),
    test("search flat runs",
      testRequest('POST', '/api/v1/runs', JSON.stringify({ projectId: sD?.workspace.projectId })),
      !!sD?.workspace.projectId,
    ),
    test(
      "search flat runs w/ hparam",
      testRequest('POST', '/api/v1/runs', JSON.stringify({
        filter: JSON.stringify({
          "filterGroup": {
            "children": [
              {
                "columnName": "hp.global_batch_size",
                "kind": "field",
                "location": "LOCATION_TYPE_RUN_HYPERPARAMETERS",
                "operator": ">",
                "type": "COLUMN_TYPE_NUMBER",
                "value": 0.1
              }
            ],
            "conjunction": "and",
            "kind": "group"
          },
          "showArchived": false
        })
      })),
      !!sD?.workspace.projectId,
    ),
  ];

  return testSuite.filter((test) => {
    if (!test.enabled && inSetupPhase)
      console.log(`SKIPPING TEST: ${test.name}`);
    return test.enabled;
  });
};

const thresholds: { [name: string]: Threshold[] } = {
  http_req_duration: [
    {
      threshold: "p(95)<100000", // Get more data before we start failing tests.
      abortOnFail: false,
    },
  ],
  http_req_failed: [
    {
      threshold: "rate<1.00", // Get more data before we start failing tests.
      abortOnFail: true,
    },
  ],
};

// In order to be able to view metrics for specific k6 groups
// we must create a unique threshold for each.
// See https://community.grafana.com/t/show-tag-data-in-output-or-summary-json-without-threshold/99320
// for more information
getloadTests(setup(true), false).forEach((group) => {
  thresholds[`http_req_duration{ group: ::${group.name}}`] = [
    {
      threshold: "p(95)<100000", // Get more data before we start failing tests.
      abortOnFail: false,
    },
  ];
});

const scenarios: { [name: string]: Scenario } = {
  average_load: {
    executor: "ramping-vus",
    stages: [
      { duration: "5m", target: 25 },
      { duration: "10m", target: 25 },
      { duration: "5m", target: 0 },
    ],
  },
};

export const options: Options = {
  scenarios,
  thresholds,
};

export default (config: TestConfiguration): void => {
  getloadTests(config, false).map((test) => test.group());
  sleep(1);
};

export function handleSummary(data: Results) {
  return {
    "junit.xml": jUnit(data, {
      name: "K6 Load Tests",
    }),
    stdout: textSummary(data),
    "results.txt": generateSlackResults(data),
  };
}
