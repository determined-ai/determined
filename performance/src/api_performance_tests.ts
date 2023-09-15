import { JSONObject, check, sleep } from "k6";
import { Options, Scenario, Threshold } from "k6/options";
import http from "k6/http";
import { jUnit, textSummary } from "./utils/k6-summary";
import {
    authenticateVU,
    generateEndpointUrl,
    test,
    testGetRequest,
} from "./utils/helpers";

const DEFAULT_CLUSTER_URL = "http://localhost:8080";

// Fallback to localhost if a cluster url is not supplied
const CLUSTER_URL = __ENV.DET_MASTER ?? DEFAULT_CLUSTER_URL;

const RBAC_ENABLED = false;

export const setup = (): TestConfiguration => {
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
    };

    const token = authenticateVU(CLUSTER_URL);
    const auth: Authorization = { token };
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

    const testSuite = [
        // Query the master endpoint
        test("get master configuration", () =>
            testGetRequest("/api/v1/master", CLUSTER_URL, testConfig)),
        test("get agents", () =>
            testGetRequest("/api/v1/agents", CLUSTER_URL, testConfig)),
        test("get workspaces", () =>
            testGetRequest("/api/v1/workspaces", CLUSTER_URL, testConfig)),
        test("get user settings", () =>
            testGetRequest("/api/v1/users/setting", CLUSTER_URL, testConfig)),
        test("get resource pools", () =>
            testGetRequest("/api/v1/resource-pools", CLUSTER_URL, testConfig)),
        test(
            "get available workspace resource pools",
            () =>
                testGetRequest(
                    `/api/v1/workspaces/${sD?.workspace.id}/available-resource-pools`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.id,
        ),
        test("get users", () =>
            testGetRequest("/api/v1/users", CLUSTER_URL, testConfig)),
        test("get workspace bindings", () =>
            testGetRequest(
                "/api/v1/resource-pools/default/workspace-bindings",
                CLUSTER_URL,
                testConfig,
            )),
        test("login", () => {
            const res = http.post(
                generateEndpointUrl("/api/v1/auth/login", CLUSTER_URL),
                JSON.stringify({
                    username: __ENV.DET_ADMIN_USERNAME ?? "admin",
                    password: __ENV.DET_ADMIN_PASSWORD ?? "",
                }),
                {
                    headers: { "Content-Type": "application/json" },
                },
            );
            check(res, { "200 response": (r) => r.status == 200 });
        }),
        test(
            "get workspace model labels",
            () =>
                testGetRequest(
                    `/api/v1/model/labels?workspaceId=${sD?.workspace.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.id,
        ),
        test("get models", () =>
            testGetRequest("/api/v1/models", CLUSTER_URL, testConfig)),
        test("get telemetry", () =>
            testGetRequest("/api/v1/master/telemetry", CLUSTER_URL, testConfig)),
        test("get tensorboards", () =>
            testGetRequest("/api/v1/tensorboards", CLUSTER_URL, testConfig)),
        test("get shells", () =>
            testGetRequest("/api/v1/shells", CLUSTER_URL, testConfig)),
        test("get notebooks", () =>
            testGetRequest("/api/v1/notebooks", CLUSTER_URL, testConfig)),
        test("get commands", () =>
            testGetRequest("/api/v1/commands", CLUSTER_URL, testConfig)),
        test("get job queue stats", () =>
            testGetRequest("/api/v1/job-queues/stats", CLUSTER_URL, testConfig)),
        test("get v2 job queue", () =>
            testGetRequest(
                "/api/v1/job-queues-v2?resourcePool=default",
                CLUSTER_URL,
                testConfig,
            )),
        test(
            "get project",
            () =>
                testGetRequest(
                    `/api/v1/projects/${sD?.workspace.projectId}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.projectId,
        ),
        test("get user activity", () =>
            testGetRequest(
                "/api/v1/user/projects/activity",
                CLUSTER_URL,
                testConfig,
            )),
        test(
            "get workspace tensorboards",
            () =>
                testGetRequest(
                    `/api/v1/tensorboards?sortBy=SORT_BY_UNSPECIFIED&orderBy=ORDER_BY_UNSPECIFIED&workspaceId=${sD?.workspace.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.id,
        ),
        test(
            "get workspace shells",
            () =>
                testGetRequest(
                    `/api/v1/shells?sortBy=SORT_BY_UNSPECIFIED&orderBy=ORDER_BY_UNSPECIFIED&workspaceId=${sD?.workspace.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.id,
        ),
        test(
            "get workspace notebooks",
            () =>
                testGetRequest(
                    `/api/v1/notebooks?limit=1000&workspaceId=${sD?.workspace.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.id,
        ),
        test(
            "get workspace commands",
            () =>
                testGetRequest(
                    `/api/v1/shells?limit=1000&workspaceId=${sD?.workspace.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.id,
        ),
        test(
            "get workspace projects",
            () =>
                testGetRequest(
                    `/api/v1/workspaces/${sD?.workspace.id}/projects`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.id,
        ),
        test("get webhooks", () =>
            testGetRequest("/api/v1/webhooks", CLUSTER_URL, testConfig)),
        test(
            "get project metric ranges",
            () =>
                testGetRequest(
                    `/api/v1/projects/${sD?.workspace.projectId}/experiments/metric-ranges`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.projectId,
        ),
        test("get project columns", () =>
            testGetRequest(`/api/v1/projects/${sD?.workspace.projectId}/columns`, CLUSTER_URL, testConfig)),
        test(
            "search experiments",
            () =>
                testGetRequest(
                    `/api/v1/experiments-search?projectId=${sD?.workspace.projectId}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.workspace.projectId,
        ),
        test("get model labels", () =>
            testGetRequest("/api/v1/model/labels", CLUSTER_URL, testConfig)),
        test(
            "get model versions",
            () =>
                testGetRequest(
                    `/api/v1/models/${sD?.model.name}/versions`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.model.name,
        ),
        test(
            "get model version",
            () =>
                testGetRequest(
                    `/api/v1/models/${sD?.model.name}/versions/${sD?.model.versionNum}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.model.versionNum && !!sD?.model.name,
        ),
        test(
            "get trial",
            () =>
                testGetRequest(
                    `/api/v1/trials/${sD?.trial.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.trial.id,
        ),
        test(
            "get trial workloads",
            () =>
                testGetRequest(
                    `/api/v1/trials/${sD?.trial.id}/workloads`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.trial.id,
        ),
        test(
            "get trial logs",
            () =>
                testGetRequest(
                    `/api/v1/trials/${sD?.trial.id}/logs/fields?`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.trial.id,
        ),
        test(
            "get experiment",
            () =>
                testGetRequest(
                    `/api/v1/experiments/${sD?.experiment.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.experiment.id,
        ),
        test(
            "get experiment metric names",
            () =>
                testGetRequest(
                    `/api/v1/experiments/metrics-stream/metric-names?ids=${sD?.experiment.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.experiment.id,
        ),
        test(
            "get experiment metric batches",
            () =>
                testGetRequest(
                    `/api/v1/experiments/${sD?.experiment.id}/metrics-stream/batches?metricName=${sD?.experiment.metricName}&metricType=${sD?.experiment.metricType}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.experiment.id &&
            !!sD?.experiment.metricName &&
            !!sD?.experiment.metricType,
        ),
        test(
            "get experiment trials sample",
            () =>
                testGetRequest(
                    `/api/v1/experiments/${sD?.experiment.id}/metrics-stream/trials-sample?metricName=${sD?.experiment.metricName}&metricType=${sD?.experiment.metricType}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.experiment.id &&
            !!sD?.experiment.metricName &&
            !!sD?.experiment.metricType,
        ),
        test(
            "get experiment trials snapshot",
            () =>
                testGetRequest(
                    `/api/v1/experiments/${sD?.experiment.id}/metrics-stream/trials-snapshot?metricName=${sD?.experiment.metricName}&metricType=${sD?.experiment.metricType}&batchesProcessed=${sD?.experiment.batches}&batchesMargin=${sD?.experiment.batchesMargin}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.experiment.id &&
            !!sD?.experiment.metricName &&
            !!sD?.experiment.metricType &&
            !!sD?.experiment.batches &&
            !!sD?.experiment.batchesMargin,
        ),
        test(
            "get experiment trials",
            () =>
                testGetRequest(
                    `/api/v1/experiments/${sD?.experiment.id}/trials`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.experiment.id,
        ),
        test(
            "get trials time series",
            () =>
                testGetRequest(
                    `/api/v1/trials/time-series?trialIds=${sD?.trial.id}&startBatches=0&metricType=METRIC_TYPE_UNSPECIFIED`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.trial.id,
        ),
        test(
            "get experiment file tree",
            () =>
                testGetRequest(
                    `/api/v1/experiments/${sD?.experiment.id}/file_tree`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.experiment.id,
        ),
        test("get experiments", () =>
            testGetRequest(
                `/api/v1/experiments?showTrialData=true`,
                CLUSTER_URL,
                testConfig,
            )),
        test("get master logs", () =>
            testGetRequest(
                `/api/v1/master/logs?offset=-1&limit=0`,
                CLUSTER_URL,
                testConfig,
            )),
        test("get resource allocations", () =>
            testGetRequest(
                `/api/v1/resources/allocation/aggregated?startDate=2000-01-01&endDate=${new Date()
                    .toJSON()
                    .slice(0, 10)}&period=RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY`,
                CLUSTER_URL,
                testConfig,
            )),
        test("get tasks", () =>
            testGetRequest(`/api/v1/tasks`, CLUSTER_URL, testConfig)),
        test("get task count", () =>
            testGetRequest(`/api/v1/tasks/count`, CLUSTER_URL, testConfig)),
        test(
            "get task",
            () =>
                testGetRequest(`/api/v1/tasks/${sD?.task.id}`, CLUSTER_URL, testConfig),
            !!sD?.task.id,
        ),
        test(
            "get task log fields",
            () =>
                testGetRequest(
                    `/api/v1/tasks/${sD?.task.id}/logs/fields`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.task.id,
        ),
        test(
            "get task logs",
            () =>
                testGetRequest(
                    `/api/v1/tasks/${sD?.task.id}/logs`,
                    CLUSTER_URL,
                    testConfig,
                ),
            !!sD?.task.id,
        ),
        test(
            "get permissions summary",
            () =>
                testGetRequest("/api/v1/permissions-summary", CLUSTER_URL, testConfig),
            RBAC_ENABLED,
        ),
        test(
            "search groups",
            () => testGetRequest("/api/groups/search", CLUSTER_URL, testConfig),
            RBAC_ENABLED,
        ),
        test(
            "get workspace roles",
            () =>
                testGetRequest(
                    `/api/v1/roles/workspace/${sD?.workspace.id}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            RBAC_ENABLED && !!sD?.workspace.id,
        ),
        test(
            "get roles by assignability",
            () =>
                testGetRequest(
                    "/api/v1/roles/search/by-assignability",
                    CLUSTER_URL,
                    testConfig,
                ),
            RBAC_ENABLED,
        ),
        test(
            "get group",
            () => testGetRequest(`/v1/groups/{groupId}`, CLUSTER_URL, testConfig),
            RBAC_ENABLED,
        ),
        test(
            "search groups",
            () => testGetRequest("/api/v1/groups/search", CLUSTER_URL, testConfig),
            RBAC_ENABLED,
        ),
        test(
            "search roles",
            () => testGetRequest(`/api/v1/roles-search`, CLUSTER_URL, testConfig),
            RBAC_ENABLED,
        ),
        test(
            "search roles by group",
            () =>
                testGetRequest(
                    `/api/v1/roles/search/by-group/{groupId}`,
                    CLUSTER_URL,
                    testConfig,
                ),
            RBAC_ENABLED,
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
            threshold: "p(95)<1000",
            abortOnFail: false,
        },
    ],
    http_req_failed: [
        {
            threshold: "rate<0.05",
            // If more than one percent of the HTTP requests fail
            // then we abort the test.
            abortOnFail: true,
        },
    ],
};

// In order to be able to view metrics for specific k6 groups
// we must create a unique threshold for each.
// See https://community.grafana.com/t/show-tag-data-in-output-or-summary-json-without-threshold/99320
// for more information
getloadTests(undefined, false).forEach((group) => {
    thresholds[`http_req_duration{ group: ::${group.name}}`] = [
        {
            threshold: "p(95)<1000",
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

export function handleSummary(data: JSONObject) {
    return {
        "junit.xml": jUnit(data, {
            name: "K6 Load Tests",
        }),
        stdout: textSummary(data),
    };
}
