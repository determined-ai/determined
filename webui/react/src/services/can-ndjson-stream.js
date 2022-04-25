"use strict";
/*exported ndjsonStream*/

var namespace = require('can-namespace');

/**
 * Brought this MIT library in to our source because...
 * 1. it doesn't seem to be an active project anymore
 * 2. there is a patch we want to apply
 * 3. it is only ~60 lines and fairly easy to maintain
 * https://github.com/canjs/can-ndjson-stream
 */
var ndjsonStream = function(response) {
  // For cancellation
  var is_reader, cancellationRequest = false;
  return new ReadableStream({
    start: function(controller) {
      var reader = response.getReader();
      is_reader = reader;
      var decoder = new TextDecoder();
      var data_buf = "";
      var errorHandler = controller.error.bind(controller);

      reader.read().then(function processResult(result) {
        if (result.done) {
          if (cancellationRequest) {
            // Immediately exit
            return;
          }

          data_buf = data_buf.trim();
          if (data_buf.length !== 0) {
            try {
              var data_l = JSON.parse(data_buf);
              controller.enqueue(data_l);
            } catch(e) {
              controller.error(e);
              return;
            }
          }
          controller.close();
          return;
        }

        var data = decoder.decode(result.value, {stream: true});
        data_buf += data;
        var lines = data_buf.split("\n");
        for(var i = 0; i < lines.length - 1; ++i) {
          var l = lines[i].trim();
          if (l.length > 0) {
            try {
              var data_line = JSON.parse(l);
              controller.enqueue(data_line);
            } catch(e) {
              controller.error(e);
              cancellationRequest = true;
              reader.cancel();
              return;
            }
          }
        }
        data_buf = lines[lines.length-1];

        return reader.read().then(processResult).catch(errorHandler);
      }).catch(errorHandler);

    },
    cancel: function(reason) {
      console.log("Cancel registered due to ", reason);
      cancellationRequest = true;
      is_reader.cancel();
    }
  });
};

module.exports = namespace.ndjsonStream = ndjsonStream;

