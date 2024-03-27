function generate(config) {
    config.logger.info('DNJ TODO making imposter: ', config.request);
    const predicatePreview = {
        exists: {
            headers: {
                'authorization': false,
                'cookie': false,
            }
        },
        matches: {
            path: config.request.path
        },
        equals: {
            method: config.request.method,
        },
        deepEquals: {
            body: config.request.body,
            query: config.request.query
        }
    };
    if (config.request.headers['authorization']) {
        predicatePreview.exists.headers['authorization'] = true;
    }
    if (config.request.headers['cookie']) {
        predicatePreview.exists.headers['cookie'] = true;
    }
    if (config.request.path.startsWith('/dynamic/http')){ // in case it's the netlify backend
        predicatePreview.matches.path = config.request.path.split('/').splice(1, 3).join('/');
    } 
    predicatePreview.matches.path = '^.*'+predicatePreview.matches.path+'$'
    const predicate = { and: [] };
    for (const [operator, matchers] of Object.entries(predicatePreview)) {
        predicate.and.push({ [operator]: matchers });
    }
    config.logger.info('DNJ TODO made imposter! ', config.request, 'Predicate: ', predicate);
    return [predicate];
}