import { JSONObject, check, sleep } from 'k6';
import { Options, Scenario, Threshold } from 'k6/options';
import http from "k6/http";
import { jUnit, textSummary } from './utils/k6-summary';
import { test, generateEndpointUrl } from './utils/helpers';


const DEFAULT_CLUSTER_URL = 'http://localhost:8080';

// Fallback to localhost if a cluster url is not supplied
const clusterURL = __ENV.DET_MASTER ?? DEFAULT_CLUSTER_URL

interface TestConfiguration {
    token: string
}

///"v2.public.eyJpZCI6MTIwODcsInVzZXJfaWQiOjEsImV4cGlyeSI6IjIwMjMtMDktMjBUMTI6MjM6MTcuODE4MS0wNTowMCJ9URiM9GD73UBazvfPNdmmbTJDV9A_pVesF1UdqVcRXQjTuxhHDu4YM-NGXFh1QMZohHc4Ef3xCxodT5iNsug6CA.bnVsbA"
export const setup = () => {
    const loginCredentials = {
        username: "admin",
        password: ""
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
    return { token }
}

// List of tests
const getloadTests = (testConfig?: TestConfiguration) => {
    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `${testConfig?.token}`,
        },
    }

    return [
        // Query the master endpoint
        test(
            'get master configuration',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/master', clusterURL));
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get agents',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/agents', clusterURL), params);
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get workspaces',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/workspaces', clusterURL), params);
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get user settings',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/users/setting', clusterURL), params);
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get resource pools',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/resource-pools', clusterURL), params);
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get available workspace resource pools',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/workspaces/1/available-resource-pools', clusterURL), params);
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get users',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/users', clusterURL), params);
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get workspace bindings',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/resource-pools/default/workspace-bindings', clusterURL), params);
                check(res, { '200 response': (r) => r.status == 200 });
            }
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
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/model/labels?workspaceId=1', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        // test(
        //     'get workspace projects',
        //     () => {
        //         const res = http.get(generateEndpointUrl('/api/v1/workspace/1/projects', clusterURL), params)
        //         check(res, { '200 response': (r) => r.status == 200 });
        //     }
        // ),
        test(
            'get models',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/models', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get telemetry',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/master/telemetry', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get tensorboards',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/tensorboards', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get shells',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/shells', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get notebooks',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/notebooks', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get commands',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/notebooks', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get job queue stats',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/job-queues/stats', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get v2 job queue',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/job-queues-2?resourcePool=default', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get project',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/projects/1', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        test(
            'get user activity',
            () => {
                const res = http.get(generateEndpointUrl('/api/v1/users/activity', clusterURL), params)
                check(res, { '200 response': (r) => r.status == 200 });
            }
        ),
        // test(
        //     'get permissions summary',
        //     () => {
        //         const res = http.get(generateEndpointUrl('/api/v1/permissions-summary', clusterURL), params);
        //         check(res, { '200 response': (r) => r.status == 200 });
        //     }
        // ),
        // test(
        //     'search groups',
        //     () => {
        //         const res = http.get(generateEndpointUrl('/api/groups/search', clusterURL), params);
        //         check(res, { '200 response': (r) => r.status == 200 });
        //     }
        // ),
        // test(
        //     'get workspace roles',
        //     () => {
        //         const res = http.get(generateEndpointUrl('/api/v1/roles/workspace/1', clusterURL), params);
        //         check(res, { '200 response': (r) => r.status == 200 });
        //     }
        // ),
        // test(
        //     'get roles by assignability',
        //     () => {
        //         const res = http.get(generateEndpointUrl('/api/v1/roles/search/by-assignability', clusterURL), params);
        //         check(res, { '200 response': (r) => r.status == 200 });
        //     }
        // ),
    ]
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
    thresholds[`http_req_duration{group: ::${group.name
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