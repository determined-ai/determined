function generate(config) {
    const predicate = {
        equals: {
            path: config.request.path
        },
        deepEquals: {
            body: config.request.body,
            query: config.request.query
        },
        exists: {
            headers: {
                'authorization': false
            }
        }
    };
    if (config.request.headers['authorization']) {
        predicate.exists.headers['authorization'] = true;
    }
    return [predicate];
}