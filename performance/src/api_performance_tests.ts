import { JSONObject, check, sleep } from 'k6';
import { Options, Scenario, Threshold } from 'k6/options';
import http from "k6/http";
import { jUnit, textSummary } from './utils/k6-summary';
import { test, generateEndpointUrl } from './utils/helpers';


const DEFAULT_CLUSTER_URL = 'http://localhost:8080';

// Fallback to localhost if a cluster url is not supplied
const clusterURL = __ENV.DET_MASTER ?? DEFAULT_CLUSTER_URL

// List of tests
const getloadTests = () => [

    // Query the master endpoint
    test(
        'visit master endpoint',
        () => {
            const res = http.get(generateEndpointUrl('/api/v1/master', clusterURL));
            check(res, { '200 response': (r) => r.status == 200 });
        }
    ),
]


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
                threshold: 'p(95)>1000',
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
            { duration: '10ms', target: 25 },
            { duration: '5m', target: 0 }
        ],
    },
}

export const options: Options = {
    scenarios,
    thresholds,
};

export default (): void => {
    getloadTests().map(test => test.group())
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