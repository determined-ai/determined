import { JSONArray, JSONObject, JSONValue, check } from 'k6';
import { SharedArray } from 'k6/data';
import { Options, Scenario } from 'k6/options';
import http from "k6/http";
import { vu } from 'k6/execution';

const clusterURL = __ENV.DET_MASTER
const masterEndpoint = '/api/v1/master'
const userEndpoint = '/api/v1/users'
const loginEndpoint = '/api/v1/auth/login'

let userVuMap: Map<any, any> = new Map();

export function setup() {
    const payload = JSON.stringify({
        username: 'admin',
        password: '',
    });

    const params: any = {
        headers: {
            'Content-Type': 'application/json',
        },
    };
    http.post(`${clusterURL}${loginEndpoint}`, payload, params);
    const userRequest = http.get(`${clusterURL}${userEndpoint}`);
    const userRequestJson = userRequest.json() as JSONObject;
    const users = userRequestJson["users"] as Array<any>
    return { users }
}

const scenarios: { [name: string]: Scenario } = {
    smoke_test: {
        tags: { test_type: 'smoke' },
        executor: 'shared-iterations',
        vus: 3,
        maxDuration: "5s",
        iterations: 5
    },
    // average_load_test: {
    //     tags: { test_type: 'average' },
    //     executor: 'ramping-vus',
    //     stages: [
    //         { duration: '10s', target: 50 },
    //         { duration: '60s', target: 50 },
    //         { duration: '10s', target: 0 }
    //     ],
    //     startTime: "1m"
    // },
    // stress_test: {
    //     tags: { test_type: 'stress' },
    //     executor: 'ramping-vus',
    //     stages: [
    //         { duration: '10s', target: 175 },
    //         { duration: '20s', target: 175 },
    //         { duration: '10s', target: 0 }
    //     ],
    //     startTime: "140s"
    // },
    // soak_test: {
    //     tags: { test_type: 'soak' },
    //     executor: 'ramping-vus',
    //     stages: [
    //         { duration: '5s', target: 50 },
    //         { duration: '1m', target: 50 },
    //         { duration: '1m', target: 0 }
    //     ],
    //     startTime: "180s"
    // },
    // spike_test: {
    //     tags: { test_type: 'spike' },
    //     executor: 'ramping-vus',
    //     stages: [
    //         { duration: '1m', target: 500 },
    //         { duration: '15s', target: 0 },
    //     ],
    //     startTime: "305s"
    // },
    // breakpoint_test: {
    //     tags: { test_type: 'breakpoint' },
    //     executor: 'ramping-arrival-rate',
    //     preAllocatedVUs: 0,
    //     stages: [
    //         { duration: '2m', target: 30000 },
    //     ],
    //     startTime: "380s",
    // },
}

export const options: Options = {
    scenarios, thresholds: {
        'http_req_failed{test_type:breakpoint}': [{
            threshold: 'rate<0.05',
            abortOnFail: true,
        },]
    },
};

export default function (data: any) {
    const vuId = vu.idInTest;
    if (!userVuMap.has(vuId)) {
        userVuMap.set(vuId, data.users[vuId])
    }
    const testUser = userVuMap.get(vuId);
    const res = http.get(`${clusterURL}${masterEndpoint}`);
    check(res, { '200 response': (r) => r.status == 200 });
}