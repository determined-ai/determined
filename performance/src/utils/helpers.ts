import { check, group, JSONObject } from "k6";
import http from "k6/http";

// k6 groups cannot be defined in the init methods of a k6 script
// this method allows us to define a group name and function
// and then return the k6 group within the test 'default' method
// the name is used to build the appropriate group thresholds.
export const test = (
    name: string,
    test_function: () => void,
    enabled: boolean = true,
): TestGroup => {
    return { name, group: () => group(name, test_function), enabled };
};

// Return the correct cluster url for a given API endpoint
export const generateEndpointUrl = (
    endpoint: string,
    clusterURL: string,
): string => `${clusterURL}${endpoint}`;

export const authenticateVU = (clusterURL: string): string => {
    if (!__ENV.DET_ADMIN_USERNAME) {
        throw new Error("Username is required")
    }
    if (!__ENV.DET_ADMIN_PASSWORD) {

    }
    const loginCredentials = {
        username: __ENV.DET_ADMIN_USERNAME,
        password: __ENV.DET_ADMIN_PASSWORD,
    };
    const params = {
        headers: { "Content-Type": "application/json" },
    };
    const requestBody = JSON.stringify(loginCredentials);
    const authResponse = http.post(
        generateEndpointUrl("/api/v1/auth/login", clusterURL),
        requestBody,
        params,
    );

    const authResponseJson = authResponse.json() as JSONObject;
    const token = `Bearer ${authResponseJson.token}`;
    return token;
};

export const testGetRequestor = (
    clusterURL: string,
    testConfig?: TestConfiguration,
): (url: string) => () => void =>
    (url: string) => {
        return () => {
            const params = {
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `${testConfig?.auth.token}`,
                },
            };
            const res = http.get(generateEndpointUrl(url, clusterURL), params);
            check(res, { "200 response": (r) => r.status == 200 });
        }
    };
