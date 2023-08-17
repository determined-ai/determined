import { check, sleep } from 'k6';
import http from "k6/http";
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

const clusterUrl = 'http://latest-main.determined.ai:8080'
const masterEndpoint = '/api/v1/master'

const scenarios = {
    smoke_test: {
        tags: { test_type: 'smoke' },
        executor: 'shared-iterations',
        vus: 5,
        iterations: 10
    },
    average_load_test: {
        tags: { test_type: 'smoke' },
        executor: 'ramping-vus',
        stages: [
            { duration: '10m', target: 100 },
            { duration: '10m', target: 0 }
        ]
    },
    // more scenarios
}

export const options = { scenarios };

export default function () {
    const res = http.get(`${clusterUrl}${masterEndpoint}`);
    check(res, { '200 response': (r) => r.status == 200 });
}

export function handleSummary(data) {
    const averageRequestDuration = data.metrics["http_req_duration"]["values"]["avg"]
    console.log(`Average master endpoint request duration is: ${averageRequestDuration / 1000} seconds`)

    return {
        stdout: textSummary(data, { enableColors: true }),
    };
}