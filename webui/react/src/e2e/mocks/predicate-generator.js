function generate(config) {
    config.logger.info('DNJ TODO making imposter: ', config.request);
    const predicatePreview = {
        exists: {
            headers: {
                'authorization': false,
                'cookie': false,
            }
        },
        equals: {
            method: config.request.method,
            path: config.request.path
        }
    };
    if (config.request.body) {
        if (!predicatePreview.deepEquals) {
            predicatePreview.deepEquals = {};
        }
        predicatePreview.deepEquals.body = config.request.body;
    }
    if (config.request.query) {
        if (!predicatePreview.deepEquals) {
            predicatePreview.deepEquals = {};
        }
        predicatePreview.deepEquals.query = config.request.query;
    }
    if (config.request.headers['authorization']) {
        predicatePreview.exists.headers['authorization'] = true;
    }
    if (config.request.headers['cookie']) {
        predicatePreview.exists.headers['cookie'] = true;
    }
    const predicate = { and: [] };
    for (const [operator, matchers] of Object.entries(predicatePreview)) {
        predicate.and.push({ [operator]: matchers });
    }
    config.logger.info('DNJ TODO made imposter! ', config.request, 'Predicate: ', predicate);
    return [predicate];
}