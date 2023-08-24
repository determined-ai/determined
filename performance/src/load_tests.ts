import { JSONObject, check } from 'k6';
import { Options, Scenario, Threshold } from 'k6/options';
import http from "k6/http";
import { jUnit, textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

const clusterURL = __ENV.DET_MASTER

const thresholds: { [name: string]: Threshold[] } = {};


// Test name and endpoint url for each test case
// the name will be used to tag relevant metrics
const tests = [{
    endpoint: '/api/v1/master',
    name: 'visit master endpoint'
},]


const scenarios: { [name: string]: Scenario } = {
    smoke: {
        executor: 'shared-iterations',
        vus: 3,
        iterations: 5
    },
    average_load: {
        executor: 'ramping-vus',
        stages: [
            { duration: '10s', target: 50 },
            { duration: '60s', target: 50 },
            { duration: '10s', target: 0 }
        ],
        startTime: "5s"
    },
    stress: {
        executor: 'ramping-vus',
        stages: [
            { duration: '10s', target: 175 },
            { duration: '20s', target: 175 },
            { duration: '10s', target: 0 }
        ],
        startTime: "90s"
    },
    soak: {
        executor: 'ramping-vus',
        stages: [
            { duration: '5s', target: 50 },
            { duration: '1m', target: 50 },
            { duration: '1m', target: 0 }
        ],
        startTime: "135s"
    },
    spike: {
        executor: 'ramping-vus',
        stages: [
            { duration: '1m', target: 500 },
            { duration: '15s', target: 0 },
        ],
        startTime: "265s"
    },
}

// In order to be able to view metrics for specific scenarios and tests
// we must create a unique threshold for each.
tests.forEach(
    (testScenario) =>
        Object.keys(scenarios).forEach((scenarioName) =>
            thresholds[`http_req_duration{test:${testScenario.name}, scenario:${scenarioName}}`] = [
                {
                    threshold: 'p(95)<1000',
                    abortOnFail: false,
                }
            ],
        )
)


export const options: Options = {
    scenarios,
    thresholds,
};

export default function (): void {
    tests.forEach((testScenario) => {
        const res = http.get(`${clusterURL}${testScenario.endpoint}`
            , {
                tags: { test: testScenario.name }
            }
        );
        check(res, { '200 response': (r) => r.status == 200 });
    })
}

export function handleSummary(data: JSONObject) {
    return {
        'junit.xml': jUnit(data, {
            name: 'K6 Load Tests'
        }),
        'stdout': textSummary(data)
    };
}