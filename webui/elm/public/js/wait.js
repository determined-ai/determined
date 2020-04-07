const ERROR_STATES = ['TERMINATED', 'TERMINATING'];

function getUrlVars() {
  let vars = {};
  let parts = window.location.href.replace(/[?&]+([^=&]+)=([^&]*)/gi, function(m, key, value) {
    vars[key] = value;
  });
  return vars;
}

const tipEl = () => document.getElementById("tip");

const spinner = () => document.getElementById("spinner");

const isFatalError = state => {
  return ERROR_STATES.includes(state);
}

function redirect(url) {
  tipEl().innerHTML = "Redirecting..."
  document.title = "Redirecting to Service";
  window.location.replace(url);
}

function msgHandler(event, waitType, readyAction) {
  console.log("Message from server ", event.data);
  let msg = JSON.parse(event.data);
  if (msg.snapshot) {
    state = msg.snapshot.state;
    document.getElementById("state").innerHTML = state;

    if (state === "RUNNING" && msg.snapshot.is_ready) {
      document.getElementById("tip").innerHTML = "Redirecting momentarily.."
      document.title = state + " - Redirecting momentarily";
      // This is a bit redundant given the code at the end of the msgHandler function,
      // but it helps avoid panics that occur in the master in the rare instances when
      // a redirect happens after the state is marked as RUNNING and before the
      // "service_ready_event" flag is raised.
      setTimeout(readyAction, 3000);
    } else if (isFatalError(state)) {
        const waitLabel = waitType === 'notebook' ? 'Notebook' : 'TensorBoard';
        document.title = state;
        spinner().className = "";
        tipEl().innerHTML =
            `The requested ${waitLabel} has been killed. Please launch a new one.`;
    } else {
        document.title = state + " - Waiting for service";
    }
  }
}

// createWsUrl: Given an event url create the corresponding ws url.
function createWsUrl(eventUrl) {
  const isFullUrl = /^https?:\/\//i;

  if (isFullUrl.test(eventUrl)) {
    return eventUrl.replace(/^http/, "ws");
  } else {
    // Remove the preceding slash if it is an absolute path.
    eventUrl = eventUrl.replace(/^\//, "");
    let url = window.location.protocol.replace(/^http/, "ws");
    url += "//" + window.location.host + "/" + eventUrl;
    return url;
  }
}

function waitForEvents(eventUrl, msgHandler, jumpDest) {
  const url = createWsUrl(eventUrl);
  const socket = new WebSocket(url);
  socket.addEventListener("open", function(event) {
    console.log("WebSocket is open:" + url);
    tipEl().innerHTML = "Waiting for service..";
  });
  socket.addEventListener("error", function(event) {
    console.error("WebSocket cannot be opened:" + url);
    tipEl().innerHTML = "Service not found";
  });
  socket.addEventListener("message", function(event) {
    const waitType = eventUrl.replace(/^\/?(notebook|tensorboard).*/i, '$1');
    msgHandler(event, waitType, function() {
      redirect(jumpDest)
    });
  });
}

(function() {
let eventUrl = decodeURIComponent(getUrlVars()["event"]);
let jumpDest = decodeURIComponent(getUrlVars()["jump"]);
console.log("eventUrl: " + eventUrl);
console.log("jumpDest: " + jumpDest);
if (typeof eventUrl !== "string" || eventUrl.length < 2) {
  console.error("Wrong or Missing 'event' Parameter in URL");
}
if (typeof jumpDest !== "string" || jumpDest.length < 2) {
  console.error("Wrong or Missing 'jump' Parameter in URL");
}
if (typeof eventUrl === "string" && eventUrl.length >= 2 && typeof jumpDest === "string" &&
    jumpDest.length >= 2) {
  waitForEvents(eventUrl, msgHandler, jumpDest)
}
})();
