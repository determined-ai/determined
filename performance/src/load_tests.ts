import { JSONObject, check, sleep } from 'k6';
import { Options, Scenario, Threshold } from 'k6/options';
import http from "k6/http";
import { jUnit, textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

const clusterURL = __ENV.DET_MASTER
const masterEndpoint = '/api/v1/master';

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

const scenarios: { [name: string]: Scenario } = {
    smoke: {
        executor: 'per-vu-iterations',
        vus: 5,
        iterations: 250
    },
}

export const options: Options = {
    scenarios,
    thresholds,
};

export default function (): void {
    const res = http.get(`${clusterURL}${masterEndpoint}`
    );
    check(res, { '200 response': (r) => r.status == 200 });
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