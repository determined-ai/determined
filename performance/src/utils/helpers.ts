import { check, group, JSONObject } from "k6";
import http from "k6/http";
import { Results, TestConfiguration, TestGroup } from "./types";

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
    throw new Error("Username is required");
  }
  if (__ENV.DET_ADMIN_PASSWORD === undefined) {
    throw new Error("Password is required");
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
): ((url: string) => () => void) => {
  return (url: string) => {
    return () => {
      const params = {
        headers: {
          "Content-Type": "application/json",
          Authorization: `${testConfig?.auth.token}`,
        },
      };
      const res = http.get(generateEndpointUrl(url, clusterURL), params);
      check(res, { "200 response": (r) => r.status == 200 });
    };
  };
};

const getTestName = (resultName: string): string =>
  "Test: " +
  resultName.substring(
    resultName.lastIndexOf(":") + 1,
    resultName.indexOf("}"),
  );

export const generateSlackResults = (results: Results): string => {
  let successfulResultString = "";
  let failureResultString = "";
  let infoString = "";
  let slackOutputString = "";
  let testFailures = 0;
  const failedRequests = `Failed HTTP Requests: ${results.metrics["http_req_failed"].values["passes"]} \n`;
  const failedRequestsPercent = `Percent Failed HTTP Requests: ${
    Number(results.metrics["http_req_failed"].values["rate"]) * 100
  }% \n`;
  const statNames = ["avg", "min", "med", "max", "p(90)", "p(95)"];
  const thresholdStatKey = "p(95)\u003c1000";
  infoString = infoString.concat(failedRequests, failedRequestsPercent);
  Object.keys(results.metrics)
    .filter((key) => key.includes("group: ::"))
    .forEach((key) => {
      const testPassed = results.metrics[key].thresholds[thresholdStatKey]?.ok;
      const stats = results.metrics[key].values;
      const groupNameString = `${getTestName(key)} \n`;
      const statsString = statNames.map(
        (name) => `${name} = ${Number(stats[name]).toFixed(2)}ms   `,
      );
      if (!testPassed) {
        testFailures++;
        failureResultString = failureResultString.concat(
          groupNameString,
          ...statsString,
          "\n\n",
        );
      } else {
        successfulResultString = successfulResultString.concat(
          groupNameString,
          ...statsString,
          "\n\n",
        );
      }
    });
  const failures = `Test Failures: ${testFailures} \n`;
  infoString = infoString.concat(failures);
  slackOutputString = slackOutputString.concat(
    infoString,
    "\n\nFailed Tests\n\n",
    failureResultString,
    "\n\nSuccessful Tests\n\n",
    successfulResultString,
  );
  return slackOutputString;
};
