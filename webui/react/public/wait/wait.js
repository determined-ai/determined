const ERROR_STATES = [ 'TERMINATED', 'TERMINATING' ];

function getUrlVars() {
  let vars = {};
  window.location.href.replace(/[?&]+([^=&]+)=([^&]*)/gi, function(m, key, value) {
    vars[key] = value;
  });
  return vars;
}

const titlePrefix = 'Determined AI:';

const tipEl = () => document.getElementById('tip');

const spinner = () => document.getElementById('spinner');

const isFatalError = state => ERROR_STATES.includes(state);

function redirect(url) {
  tipEl().innerHTML = 'Redirecting...';
  document.title = `${titlePrefix} Redirecting to Service`;
  window.location.replace(url);
}

function msgHandler(event, waitType, readyAction) {
  let msg = JSON.parse(event.data);
  console.log('Message from server ', msg);
  if (msg.snapshot) {
    const state = msg.snapshot.state;
    document.getElementById('state').innerHTML = state;

    if (state === 'RUNNING' && msg.snapshot.is_ready) {
      document.getElementById('tip').innerHTML = 'Redirecting momentarily..';
      document.title = `${titlePrefix} ${state} - Redirecting momentarily`;
      readyAction();
    } else if (isFatalError(state)) {
      const waitLabel = waitType === 'notebook' ? 'Notebook' : 'TensorBoard';
      document.title = `${titlePrefix} ${state}`;
      spinner().className = '';
      tipEl().innerHTML = `The requested ${waitLabel} has been killed. Please launch a new one.`;
    } else {
      document.title = `${titlePrefix} ${state} - Waiting for service`;
    }
  }
}

// createWsUrl: Given an event url create the corresponding ws url.
function createWsUrl(eventUrl) {
  const isFullUrl = /^https?:\/\//i;

  if (isFullUrl.test(eventUrl)) {
    return eventUrl.replace(/^http/, 'ws');
  } else {
    // Remove the preceding slash if it is an absolute path.
    eventUrl = eventUrl.replace(/^\//, '');
    let url = window.location.protocol.replace(/^http/, 'ws');
    url += '//' + window.location.host + '/' + eventUrl;
    return url;
  }
}

function waitForEvents(eventUrl,jumpDest) {
  const url = createWsUrl(eventUrl);
  const socket = new WebSocket(url);
  socket.addEventListener('open', function() {
    console.log(`WebSocket is open: ${url}`);
    tipEl().innerHTML = 'Waiting for service..';
  });
  socket.addEventListener('error', function() {
    console.error(`WebSocket cannot be opened: ${url}`);
    tipEl().innerHTML = 'Service not found';
  });
  socket.addEventListener('message', function(event) {
    const waitType = eventUrl.replace(/^\/?(notebook|tensorboard).*/i, '$1');
    msgHandler(event, waitType, function() {
      redirect(jumpDest);
    });
  });
}

(function() {
  let eventUrl = decodeURIComponent(getUrlVars()['event']);
  let jumpDest = decodeURIComponent(getUrlVars()['jump']);
  console.log(`eventUrl: ${eventUrl}`);
  console.log(`jumpDest: ${jumpDest}`);
  if (typeof eventUrl !== 'string' || eventUrl.length < 2) {
    console.error("Wrong or Missing 'event' Parameter in URL");
  }
  if (typeof jumpDest !== 'string' || jumpDest.length < 2) {
    console.error("Wrong or Missing 'jump' Parameter in URL");
  }
  if (typeof eventUrl === 'string' && eventUrl.length >= 2 &&
      typeof jumpDest === 'string' && jumpDest.length >= 2) {
    waitForEvents(eventUrl, jumpDest);
  }
})();
