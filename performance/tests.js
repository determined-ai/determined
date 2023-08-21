import { check, sleep } from 'k6';
import http from "k6/http";
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

const clusterUrl = 'http://latest-main.determined.ai:8080'
const masterEndpoint = '/api/v1/master'

const scenarios = {
    smoke_test: {
        tags: { test_type: 'smoke' },
        executor: 'shared-iterations',
        vus: 3,
        maxDuration: "1m",
        iterations: 5
    },
    average_load_test: {
        tags: { test_type: 'average' },
        executor: 'ramping-vus',
        stages: [
            { duration: '10m', target: 50 },
            { duration: '60m', target: 50 },
            { duration: '10m', target: 0 }
        ],
        startTime: "1m"
    },
    stress_test: {
        tags: { test_type: 'stress' },
        executor: 'ramping-vus',
        stages: [
            { duration: '10m', target: 175 },
            { duration: '20m', target: 175 },
            { duration: '10m', target: 0 }
        ],
        startTime: "85m"
    },
    soak_test: {
        tags: { test_type: 'soak' },
        executor: 'ramping-vus',
        stages: [
            { duration: '5m', target: 50 },
            { duration: '1h', target: 50 },
            { duration: '10m', target: 0 }
        ],
        startTime: "130m"
    },
    spike_test: {
        tags: { test_type: 'spike' },
        executor: 'ramping-vus',
        stages: [
            { duration: '3m', target: 500 },
            { duration: '1m', target: 0 },
        ],
        startTime: "210m"
    },
    breakpoint_test: {
        tags: { test_type: 'breakpoint' },
        executor: 'ramping-arrival-rate',
        preAllocatedVUs: 0,
        stages: [
            { duration: '2h', target: 30000 },
        ],
        startTime: "215m",
    },
}

export const options = {
    scenarios, thresholds: {
        'http_req_failed{test_type:breakpoint}': [{
            threshold: 'rate<0.05',
            abortOnFail: true,
        },]
    },
};

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