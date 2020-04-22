// jshint esversion: 6


function dispatchResize(target, width, height) {
  // Elements can get resized to 0 when the logical page changes and they disappear; dispatching
  // that event breaks some internal invariant of Elm and causes runtime errors. Ensuring that the
  // target is in the DOM prevents that from happening.
  if (document.contains(target)) {
    target.dispatchEvent(new CustomEvent("resize", {detail: {width: width, height: height}}));
  }
}


// Define a custom <resize-monitor> element that emits resize events when its size changes.
class ResizeMonitor extends HTMLElement {
  constructor() {
    super();
    new ResizeObserver((entries) => {
      for (const entry of entries) {
        const {width, height} = entry.contentRect;
        dispatchResize(entry.target, width, height);
      }
    }).observe(this);
  }
}


customElements.define("resize-monitor", ResizeMonitor);


let DetShims = function() {
  function addPorts(app) {
    addAceEditorPorts(app);
    addExitFullscreenPort(app)
    addJumpToPointPort(app);
    addKickResizePort(app);
    addOpenNewWindowPort(app);
    addRequestFullscreenPort(app);
    addClipboardPorts(app);
    addSetPageTitlePort(app);
    addAssignLocationPort(app);
    addSegmentAnalyticsPorts(app);
  }

  let aceEditors = {};

  function addAceEditorPorts(app) {
    app.ports.setUpAceEditor.subscribe((args) => {
      // annotateError sets an Ace annotation to indicate to the user the location of a syntax
      // error.
      let annotateError = (editor, error) => {
        const mark = error.mark;
        editor.getSession().setAnnotations(
            [{row: mark.line, column: mark.column, type: "error", text: "Syntax error"}]);
      };

      let doSetup = () => {
        const id = args[0];
        const initialValue = args[1];
        const editor = ace.edit(id);

        let changeHandler = () => {
          editor.getSession().clearAnnotations();
          const content = editor.getValue();
          let badSyntax = false;

          try {
            jsyaml.safeLoad(content);
          } catch (error) {
            annotateError(editor, error);
            badSyntax = true;
          }

          // Send new content to Elm.
          app.ports.aceContentUpdated.send({content: content, badSyntax: badSyntax});
        };

        editor.on("change", changeHandler);

        // TODO(jgevirtz): This should probably be configurable via Elm.
        editor.setTheme("ace/theme/xcode");
        editor.session.setMode("ace/mode/yaml");
        editor.setOptions({showLineNumbers: false});
        editor.setAutoScrollEditorIntoView(true);
        editor.setValue(initialValue, 0);
        editor.clearSelection();

        // In case the initial value contains incorrect yaml syntax:
        let badSyntax = false;
        try {
          jsyaml.safeLoad(initialValue);
        } catch (error) {
          annotateError(editor, error);
          badSyntax = true;
        }

        // Send content to Elm to make sure it is aware of any syntax errors that were detected by
        // jsyaml.
        app.ports.aceContentUpdated.send({content: initialValue, badSyntax: badSyntax});

        aceEditors[id] = editor;
      };
      requestAnimationFrame(doSetup);
    });

    app.ports.resizeAceEditor.subscribe((args) => {
      requestAnimationFrame( () => {
        const id = args;
        let editor = aceEditors[id];
        if (editor) {
          editor.resize(true);
        } else {
          console.error(`Can not find editor ${id}`);
        }
      });
    });

    app.ports.destroyAceEditor.subscribe((id) => {
      const editor = aceEditors[id];
      if (!editor) {
        // TODO(jgevirtz): Report error to Elm?
        return;
      }

      editor.destroy();

      aceEditors[id] = null;
    });
  }

  function addExitFullscreenPort(app) {
    app.ports.exitFullscreenPort.subscribe(() => {
      document.exitFullscreen();
    });
  }

  function addJumpToPointPort(args) {
    app.ports.jumpToPointPort.subscribe(([id, pos]) => {
      requestAnimationFrame(() => {
        const e = document.getElementById(id);
        if (e) {
          e.scrollTop = e.scrollHeight - pos;
        };
      });
    });
  }

  function addKickResizePort(app) {
    // Make all <resize-monitor> elements report their current size. This is a bit of a hack to work
    // around Virtual DOM's reuse of nodes, which can otherwise cause events to be dropped if the
    // same element is used in two different logical pages and ends up being the same size in both.
    // At the time of writing, this is not strictly necessary because special views for loading mean
    // that no <resize-monitor> element can stay present across a page transition, but that could
    // change in the future.
    let doKick = () => {
      document.querySelectorAll("resize-monitor").forEach((d) => {
        let {width, height} = d.getBoundingClientRect();
        dispatchResize(d, width, height);
      });
    };
    app.ports.kickResizePort.subscribe(() => {
      requestAnimationFrame(doKick);
    });
  }

  function addOpenNewWindowPort(app) {
    app.ports.openNewWindowPort.subscribe((url) => {
      window.open(url);
    });
  }

  function addRequestFullscreenPort(app) {
    app.ports.requestFullscreenPort.subscribe((id) => {
      document.getElementById(id).requestFullscreen();
    });
  }

  function addSetPageTitlePort(app) {
    app.ports.setPageTitle.subscribe((title) => {
      document.title = title;
    });
  }

  function addAssignLocationPort(app) {
    app.ports.assignLocation.subscribe((uri) => {
      document.location.assign(uri);
    });
  }

  function addClipboardPorts(app) {
    app.ports.copyToClipboard.subscribe( id => {
      const element = document.getElementById(id);
      let success = false;

      if (element) {
        const range = document.createRange();
        // Add all the text in the component to this range.
        range.selectNode(element);

        const selection = window.getSelection();
        // If the user has already selected something, we save it so we can restore it when we are
        // done.
        let originalSelection = 
          selection.rangeCount > 0 ? selection.getRangeAt(0) : null;
        
        // NoOp if there is nothing selected, but doesn't hurt anything.
        selection.removeAllRanges();
        
        // Select the text in the component.
        selection.addRange(range);
        // Copy.
        document.execCommand("copy");

        // Remove our selection.
        selection.removeAllRanges();

        // If the user had anything selected previously, restore it.
        if (originalSelection) {
          selection.addRange(originalSelection);  
        }
        
        // window.getSelection().removeRange(range);
        success = true;
      } 

      if (app.ports.copiedToClipboard) {
        app.ports.copiedToClipboard.send(success);
      }
    });
  }

  function addSegmentAnalyticsPorts(app) {
    app.ports.loadAnalytics.subscribe((segmentKey) => {
      if (!window.analytics) return;
      window.analytics.load(segmentKey);
      window.analytics.page();
    });

    app.ports.setAnalyticsIdentityPort.subscribe((clusterId) => {
      if (!window.analytics) return;
      window.analytics.identify(clusterId);
    });

    app.ports.setAnalyticsPagePort.subscribe((pathname) => {
      if (!window.analytics) return;
      window.analytics.page(pathname);
    });

    // For future use. Uncommenting will cause errors until it is actually used.
    // app.ports.setAnalyticsEventPort.subscribe((args) => {
    //   if (!window.analytics) return;
    //   window.analytics.track(args[0], JSON.parse(args[1]));
    // });
  }

  return {addPorts: addPorts};
}();
