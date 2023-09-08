import { group } from 'k6';

// k6 groups cannot be defined in the init methods of a k6 script
// this method allows us to define a group name and function
// and then return the k6 group within the test 'default' method
// the name is used to build the appropriate group thresholds.
export const test = (name: string, test_function: () => unknown) => {
    return { name, group: () => group(name, test_function) }
}

// Return the correct cluster url for a given API endpoint
export const generateEndpointUrl = (endpoint: string, clusterURL: string) =>
    `${clusterURL}${endpoint}`