import { JSONArray, JSONObject, JSONValue, check, sleep } from 'k6';
import { Options, Scenario, Threshold } from 'k6/options';
import http from "k6/http";
import { jUnit, textSummary } from './utils/k6-summary';
import { test, generateEndpointUrl } from './utils/helpers';

const DEFAULT_CLUSTER_URL = 'http://localhost:8080';

// Fallback to localhost if a cluster url is not supplied
const clusterURL = __ENV.DET_MASTER ?? DEFAULT_CLUSTER_URL

const RBAC_ENABLED = false;

interface Model {
    name: string
    versionNum: number
}

interface Trial {
    id: number
}

interface Workspace {
    id: number
}

interface Experiment {
    id: number
    metricName: string
    metricType: string
    batches: number
    batchesMargin: number
}
interface SeededData {
    model: Model
    trial: Trial
    experiment: Experiment
    workspace: Workspace
}
interface Authorization {
    token: string
}
interface TestConfiguration {
    auth: Authorization,
    seededData: SeededData
}

interface TestGroup {
    name: string
    group: () => void;
    rbacRequired: boolean;
}


const authenticateVU = () => {
    const loginCredentials = {
        username: __ENV.DET_ADMIN_USERNAME,
        password: __ENV.DET_ADMIN_PASSWORD
    }
    const params = {
        headers: { 'Content-Type': 'application/json' },
    }
    const requestBody = JSON.stringify(loginCredentials)
    const authResponse = http.post(generateEndpointUrl('/api/v1/auth/login', clusterURL),
        requestBody,
        params)

    const authResponseJson = authResponse.json() as JSONObject
    const token = `Bearer ${authResponseJson.token}`;
    return token
}

export const setup = (): TestConfiguration => {
    const model = {
        name: 'Test model',
        versionNum: 1
    }

    const trial = {
        id: 352
    }

    const workspace = {
        id: 352
    }

    const experiment = {
        id: 10951,
        metricName: 'validation_error',
        metricType: 'METRIC_TYPE_VALIDATION',
        batches: 1000,
        batchesMargin: 100
    }
    const seededData: SeededData = {
        model,
        trial,
        experiment,
        workspace
    };

    const token = authenticateVU()
    const auth: Authorization = { token }
    const testConfig: TestConfiguration = { auth, seededData }
    return testConfig
}

const testGetRequest = (url: string, clusterURL: string, testConfig?: TestConfiguration,) => {
    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `${testConfig?.auth.token}`,
        },
    }
    const res = http.get(generateEndpointUrl(url, clusterURL), params);
    check(res, { '200 response': (r) => r.status == 200 });
}

// List of tests
const getloadTests = (testConfig?: TestConfiguration): TestGroup[] => {

    const testSuite = [
        // Query the master endpoint
        test(
            'get master configuration',
            () => testGetRequest('/api/v1/master', clusterURL, testConfig)
        ),
        test(
            'get agents',
            () => testGetRequest('/api/v1/agents', clusterURL, testConfig)
        ),
        test(
            'get workspaces',
            () => testGetRequest('/api/v1/workspaces', clusterURL, testConfig)
        ),
        test(
            'get user settings',
            () => testGetRequest('/api/v1/users/setting', clusterURL, testConfig)
        ),
        test(
            'get resource pools',
            () => testGetRequest('/api/v1/resource-pools', clusterURL, testConfig)
        ),
        test(
            'get available workspace resource pools',
            () => testGetRequest('/api/v1/workspaces/1/available-resource-pools', clusterURL, testConfig)
        ),
        test(
            'get users',
            () => testGetRequest('/api/v1/users', clusterURL, testConfig)
        ),
        test(
            'get workspace bindings',
            () => testGetRequest('/api/v1/resource-pools/default/workspace-bindings', clusterURL, testConfig)
        ),
        test(
            'login',
            () => {
                const res = http.post(generateEndpointUrl('/api/v1/auth/login', clusterURL),
                    JSON.stringify({
                        username: "admin",
                        password: ""
                    }),
                    {
                        headers: { 'Content-Type': 'application/json' },
                    })
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get workspace model labels',
            () => testGetRequest('/api/v1/model/labels?workspaceId=1', clusterURL, testConfig)
        ),
        test(
            'get models',
            () => testGetRequest('/api/v1/models', clusterURL, testConfig)
        ),
        test(
            'get telemetry',
            () => testGetRequest('/api/v1/master/telemetry', clusterURL, testConfig)
        ),
        test(
            'get tensorboards',
            () => testGetRequest('/api/v1/tensorboards', clusterURL, testConfig)
        ),
        test(
            'get shells',
            () => testGetRequest('/api/v1/shells', clusterURL, testConfig)
        ),
        test(
            'get notebooks',
            () => testGetRequest('/api/v1/notebooks', clusterURL, testConfig)
        ),
        test(
            'get commands',
            () => testGetRequest('/api/v1/commands', clusterURL, testConfig)
        ),
        test(
            'get job queue stats',
            () => testGetRequest('/api/v1/job-queues/stats', clusterURL, testConfig)
        ),
        test(
            'get v2 job queue',
            () => testGetRequest('/api/v1/job-queues-v2?resourcePool=default', clusterURL, testConfig)
        ),
        test(
            'get project',
            () => testGetRequest('/api/v1/projects/1', clusterURL, testConfig)
        ),
        test(
            'get user activity',
            () => testGetRequest('/api/v1/user/projects/activity', clusterURL, testConfig)
        ),
        test(
            'get workspace tensorboards',
            () => testGetRequest('/api/v1/tensorboards?sortBy=SORT_BY_UNSPECIFIED&orderBy=ORDER_BY_UNSPECIFIED&workspaceId=1', clusterURL, testConfig)
        ),
        test(
            'get workspace shells',
            () => testGetRequest('/api/v1/shells?sortBy=SORT_BY_UNSPECIFIED&orderBy=ORDER_BY_UNSPECIFIED&workspaceId=1', clusterURL, testConfig)
        ),
        test(
            'get workspace notebooks',
            () => testGetRequest('/api/v1/notebooks?limit=1000&workspaceId=1', clusterURL, testConfig)
        ),
        test(
            'get workspace commands',
            () => testGetRequest('/api/v1/shells?limit=1000&workspaceId=1', clusterURL, testConfig)
        ),
        test(
            'get workspace projects',
            () => testGetRequest('/api/v1/workspaces/1/projects', clusterURL, testConfig)
        ),
        test(
            'get webhooks',
            () => testGetRequest('/api/v1/webhooks', clusterURL, testConfig)
        ),
        test(
            'get project metric ranges',
            () => testGetRequest('/api/v1/projects/1/experiments/metric-ranges', clusterURL, testConfig)
        ),
        test(
            'get project columns',
            () => testGetRequest('/api/v1/projects/1/columns', clusterURL, testConfig)
        ),
        test(
            'search experiments',
            () => testGetRequest('/api/v1/experiments-search?projectId=1', clusterURL, testConfig)
        ),
        test(
            'get model labels',
            () => testGetRequest('/api/v1/model/labels', clusterURL, testConfig)
        ),
        test(
            'get model versions',
            () => testGetRequest(`/api/v1/models/${testConfig?.seededData.model.name}/versions`, clusterURL, testConfig)
        ),
        test(
            'get model version',
            () => testGetRequest(`/api/v1/models/${testConfig?.seededData.model.name}/versions/${testConfig?.seededData.model.versionNum}`, clusterURL, testConfig)
        ),
        test(
            'get trial',
            () => testGetRequest(`/api/v1/trials/${testConfig?.seededData.trial.id}`, clusterURL, testConfig)
        ),
        test(
            'get trial workloads',
            () => testGetRequest(`/api/v1/trials/${testConfig?.seededData.trial.id}/workloads`, clusterURL, testConfig)
        ),
        test(
            'get trial logs',
            () => testGetRequest(`/api/v1/trials/${testConfig?.seededData.trial.id}/logs/fields?`, clusterURL, testConfig)
        ),
        test(
            'get experiment',
            () => testGetRequest(`/api/v1/experiments/${testConfig?.seededData.experiment.id}`, clusterURL, testConfig)
        ),
        test(
            'get experiment metric names',
            () => testGetRequest(`/api/v1/experiments/metrics-stream/metric-names?ids=${testConfig?.seededData.experiment.id}`, clusterURL, testConfig)),
        test(
            'get experiment metric batches',
            () => testGetRequest(`/api/v1/experiments/${testConfig?.seededData.experiment.id}/metrics-stream/batches?metricName=${testConfig?.seededData.experiment.metricName}&metricType=${testConfig?.seededData.experiment.metricType}`, clusterURL, testConfig)
        ),
        test(
            'get experiment trials sample',
            () => testGetRequest(`/api/v1/experiments/${testConfig?.seededData.experiment.id}/metrics-stream/trials-sample?metricName=${testConfig?.seededData.experiment.metricName}&metricType=${testConfig?.seededData.experiment.metricType}`, clusterURL, testConfig)
        ),
        test(
            'get experiment trials snapshot',
            () => testGetRequest(`/api/v1/experiments/${testConfig?.seededData.experiment.id}/metrics-stream/trials-snapshot?metricName=${testConfig?.seededData.experiment.metricName}&metricType=${testConfig?.seededData.experiment.metricType}&batchesProcessed=${testConfig?.seededData.experiment.batches}&batchesMargin=${testConfig?.seededData.experiment.batchesMargin}`, clusterURL, testConfig)
        ),
        test(
            'get experiment trials',
            () => testGetRequest(`/api/v1/experiments/${testConfig?.seededData.experiment.id}/trials`, clusterURL, testConfig)
        ),
        test(
            'get trials time series',
            () => testGetRequest(`/api/v1/trials/time-series?trialIds=${testConfig?.seededData.trial.id}&startBatches=0&metricType=METRIC_TYPE_UNSPECIFIED`, clusterURL, testConfig)
        ),
        test(
            'get experiment file tree',
            () => testGetRequest(`/api/v1/experiments/${testConfig?.seededData.experiment.id}/file_tree`, clusterURL, testConfig)
        ),
        test(
            'get experiments',
            () => testGetRequest(`/api/v1/experiments?showTrialData=true`, clusterURL, testConfig)
        ),
        test(
            'get master logs',
            () => testGetRequest(`/api/v1/master/logs?offset=-1&limit=0`, clusterURL, testConfig)
        ),
        test(
            'get resource allocations',
            () => testGetRequest(`/api/v1/resources/allocation/aggregated?startDate=2000-01-01&endDate=${new Date().toJSON().slice(0, 10)}&period=RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY`, clusterURL, testConfig)
        ),
        test(
            'get tasks',
            () => testGetRequest(`/api/v1/tasks`, clusterURL, testConfig)
        ),
        test(
            'get permissions summary',
            () => testGetRequest('/api/v1/permissions-summary', clusterURL, testConfig),
            true
        ),
        test(
            'search groups',
            () => testGetRequest('/api/groups/search', clusterURL, testConfig),
            true
        ),
        test(
            'get workspace roles',
            () => testGetRequest('/api/v1/roles/workspace/1', clusterURL, testConfig),
            true
        ),
        test(
            'get roles by assignability',
            () => testGetRequest('/api/v1/roles/search/by-assignability', clusterURL, testConfig),
            true
        ),
        test(
            'get group',
            () => testGetRequest(`/v1/groups/{groupId}`, clusterURL, testConfig),
            true
        ),
        test(
            'search groups',
            () => testGetRequest('/api/v1/groups/search', clusterURL, testConfig),
            true
        ),
        test(
            'search roles',
            () => testGetRequest(`/api/v1/roles-search`, clusterURL, testConfig),
            true
        ),
        test(
            'search roles by group',
            () => testGetRequest(`/api/v1/roles/search/by-group/{groupId}`, clusterURL, testConfig),
            true
        ),
    ]

    return testSuite.filter((suite) => !suite.rbacRequired || RBAC_ENABLED)
}


const thresholds: { [name: string]: Threshold[] } = {
    http_req_duration: [
        {
            threshold: 'p(95)<1000',
            abortOnFail: false,
        }
    ],
    http_req_failed: [
        {
            threshold: 'rate<0.01',
            // If more than one percent of the HTTP requests fail
            // then we abort the test.
            abortOnFail: true,
        }
    ],
};

// In order to be able to view metrics for specific k6 groups
// we must create a unique threshold for each.
// See https://community.grafana.com/t/show-tag-data-in-output-or-summary-json-without-threshold/99320 
// for more information
getloadTests().forEach((group) => {
    thresholds[`http_req_duration{ group: :: ${group.name
        }}`] = [
            {
                threshold: 'p(95)<1000',
                abortOnFail: false,
            }
        ]
}
)

const scenarios: { [name: string]: Scenario } = {
    average_load: {
        executor: 'ramping-vus',
        stages: [
            { duration: '5m', target: 25 },
            { duration: '10m', target: 25 },
            { duration: '5m', target: 0 }
        ],
    },
}

export const options: Options = {
    scenarios,
    thresholds,
};

export default (config: TestConfiguration): void => {
    getloadTests(config).map(test => test.group())
    sleep(1)
}

export function handleSummary(data: JSONObject) {
    return {
        'junit.xml': jUnit(data, {
            name: 'K6 Load Tests'
        }),
        'stdout': textSummary(data)
    };
}