function userMock(config) {
    class User {
        constructor(username, displayName, admin = false, active = true, remote = false, agentUserGroup = null) {
            this.id = Math.floor(Math.random() * 10000000);
            this.username = username;
            this.admin = admin;
            this.active = active;
            this.agentUserGroup = null;
            this.displayName = displayName;
            this.modifiedAt = new Date().getTime();
            this.remote = false;
        }
    }
    let resp = {}
    config.logger.info('REQUEST: ' + JSON.stringify(config.request));
    const method = config.request.method;
    if (method === 'POST') {
        const body = JSON.parse(config.request.body).user;
        const user = new User(
            body.username,
            body.displayName,
            body.admin,
            body.active,
            body.remote,
            body.agentUserGroup
        );
        resp = {
            statusCode: 200, // DNJ TODO - invalid no password
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 'user': user })
        };
        config.state.users = {};
        config.state.users[user.username] = user;
    }else if(method === 'PATCH'){
        if (!cache.state.users){
            config.callback({statusCode: 404});
            return;
        }
        const body = JSON.parse(config.request.body).user;
        const existingUser = cache.state.users[body.username]
        for (const [key, value] of Object.entries(body)){
            existingUser[key]=value;
        }
        resp = {
            statusCode: 200, // DNJ TODO - invalid no password
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 'user': existingUser })
        };
    }else if(method === 'GET'){

    }
    config.logger.info('Successfully proxied: ' + JSON.stringify(resp));
    config.logger.info('Current user cache state: ' + JSON.stringify(config.state.users));
    config.callback(resp);
}