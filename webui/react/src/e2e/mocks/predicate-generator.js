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
     // lots of headers we don't care about. Especially constantly changing things 
     // like Date, so each one gets special handling. 
    if (config.request.headers['authorization']) {
        predicatePreview.exists.headers['authorization'] = true;
    }
    if (config.request.headers['cookie']) {
        predicatePreview.exists.headers['cookie'] = true;
    }
    if (config.request.path.startsWith('/dynamic/http')){ // in case it's the netlify backend
        // we need to strip the netlify path from the matcher in format /dynamic/http/0.0.0.0:1234/my/real/path
        // down to /my/real/path so we can match without netlify
        let path=config.request.path.split('/')
        path.splice(1,3)
        predicatePreview.matches.path = path.join('/');
    } 
    predicatePreview.matches.path = '^.*'+predicatePreview.matches.path+'$'
    const predicate = { and: [] }; // we AND all of our matchers so we can exact match each request
    for (const [operator, matchers] of Object.entries(predicatePreview)) {
        predicate.and.push({ [operator]: matchers });
    }
    config.logger.info('DNJ TODO made imposter! ', config.request, 'Predicate path: ', predicatePreview.matches.path);
    return [predicate];
}
